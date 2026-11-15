// Package napr reads the parcels and the addresses from the NAPR client and converts them
// to the domain. It is the anti-corruption layer of the context for NAPR.
package napr

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	naprclient "github.com/abgeo/georgia-mcp/pkg/napr"
)

const (
	parcelLabelPrefix = "lr_parcels:"
	addressInfoPath   = "/ar/mg/getinfo"
	documentDate      = "02.01.2006"
	mapPage           = naprclient.DefaultBaseURL + "/map/portal/#search/result/getinfo?lbl="
)

var (
	errNoDetail    = errors.New("no detail of a parcel that the search found")
	errOtherCode   = errors.New("a detail of another code")
	errNoShape     = errors.New("no shape")
	errNoLabel     = errors.New("a result with no label")
	errBadDistance = errors.New("a distance that is not a number")
)

// Adapter implements app.ParcelSource with the NAPR client.
type Adapter struct {
	client *naprclient.Client
	now    func() time.Time
}

var _ app.ParcelSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *naprclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Parcel finds the label of the code, then reads the detail and the shape concurrently.
func (a *Adapter) Parcel(ctx context.Context, code domain.CadastralCode) (app.ParcelRead, error) {
	results, err := a.client.Search(ctx, code.String())
	if err != nil {
		return app.ParcelRead{}, unavailable(err)
	}

	label, found := parcelLabel(results, code)
	if !found {
		return app.ParcelRead{}, fmt.Errorf("searching %s: %w", code, app.ErrParcelNotFound)
	}

	var (
		infos             []naprclient.ParcelInfo
		shapes            []naprclient.Shape
		infoErr, shapeErr error
		group             sync.WaitGroup
	)

	group.Go(func() { infos, infoErr = a.client.Parcel(ctx, label) })
	group.Go(func() { shapes, shapeErr = a.client.ParcelShape(ctx, label) })
	group.Wait()

	if err := errors.Join(infoErr, shapeErr); err != nil {
		return app.ParcelRead{}, unavailable(err)
	}

	parcel, err := toParcel(code, infos, shapes)
	if err != nil {
		return app.ParcelRead{}, unavailable(err)
	}

	return app.ParcelRead{Parcel: parcel, MapURL: mapPage + label, ReadAt: a.readAt()}, nil
}

// ParcelsNear reads the parcels near the point, in the order of NAPR.
func (a *Adapter) ParcelsNear(ctx context.Context, point domain.Point) (app.NearbyRead, error) {
	items, err := a.client.ParcelsAt(ctx, point.Lat, point.Lon)
	if err != nil {
		return app.NearbyRead{}, unavailable(err)
	}

	parcels := make([]domain.NearbyParcel, 0, len(items))

	for _, item := range items {
		parcel, err := toNearby(item)
		if err != nil {
			return app.NearbyRead{}, unavailable(err)
		}

		parcels = append(parcels, parcel)
	}

	return app.NearbyRead{Parcels: parcels, ReadAt: a.readAt()}, nil
}

// MatchAddresses reads the addresses that match the text. A parcel in the answer is left
// out.
func (a *Adapter) MatchAddresses(ctx context.Context, text string) (app.MatchRead, error) {
	results, err := a.client.Search(ctx, text)
	if err != nil {
		return app.MatchRead{}, unavailable(err)
	}

	matches := make([]app.AddressMatch, 0, len(results))

	for _, result := range results {
		if !strings.HasPrefix(result.Details.InfoLink, addressInfoPath) {
			continue
		}

		label, ok := naprclient.Label(result.Details.InfoLink)
		if !ok {
			return app.MatchRead{}, unavailable(fmt.Errorf("%w: %q", errNoLabel, result.Name))
		}

		matches = append(matches, app.AddressMatch{
			Text:         domain.CollapseSpaces(result.Name),
			Municipality: domain.CollapseSpaces(result.Descript),
			Ref:          app.AddressRef(label),
		})
	}

	return app.MatchRead{Matches: matches, ReadAt: a.readAt()}, nil
}

// AddressCentre reads the shape of the address and returns its centre.
func (a *Adapter) AddressCentre(ctx context.Context, ref app.AddressRef) (app.CentreRead, error) {
	shapes, err := a.client.AddressShape(ctx, string(ref))
	if err != nil {
		return app.CentreRead{}, unavailable(err)
	}

	point, err := shapeCentre(shapes)
	if err != nil {
		return app.CentreRead{}, unavailable(err)
	}

	return app.CentreRead{Centre: point, ReadAt: a.readAt()}, nil
}

func (a *Adapter) readAt() time.Time {
	return a.now().In(kernel.TbilisiZone())
}

func parcelLabel(results []naprclient.SearchResult, code domain.CadastralCode) (string, bool) {
	for _, result := range results {
		if strings.TrimSpace(result.Name) != code.String() {
			continue
		}

		label, ok := naprclient.Label(result.Details.InfoLink)
		if ok && strings.HasPrefix(label, parcelLabelPrefix) {
			return label, true
		}
	}

	return "", false
}

func toParcel(
	code domain.CadastralCode,
	infos []naprclient.ParcelInfo,
	shapes []naprclient.Shape,
) (domain.Parcel, error) {
	if len(infos) == 0 {
		return domain.Parcel{}, errNoDetail
	}

	detail := infos[0].Detail

	got, err := domain.ParseCadastralCode(detail.CadCode)
	if err != nil || got != code {
		return domain.Parcel{}, fmt.Errorf("%w: %q", errOtherCode, detail.CadCode)
	}

	area, err := domain.NewArea(detail.AreaMeasured, detail.Unit)
	if err != nil {
		return domain.Parcel{}, fmt.Errorf("reading the area of %s: %w", code, err)
	}

	documents, err := toDocuments(infos[0].Docs)
	if err != nil {
		return domain.Parcel{}, err
	}

	point, err := shapeCentre(shapes)
	if err != nil {
		return domain.Parcel{}, err
	}

	return domain.Parcel{
		Code:      code,
		Address:   domain.CollapseSpaces(detail.Address),
		Area:      area,
		LandType:  domain.ParseLandType(detail.LandType),
		Ownership: domain.ParseOwnershipType(detail.OwnTypeName),
		Owners:    toOwners(infos[0].Owners),
		Documents: domain.SortDocuments(documents),
		Centre:    point,
	}, nil
}

func toOwners(owners []naprclient.Owner) []string {
	names := make([]string, 0, len(owners))

	for _, owner := range owners {
		if name := domain.CollapseSpaces(owner.Person); name != "" {
			names = append(names, name)
		}
	}

	return names
}

func toDocuments(sets []naprclient.DocumentSet) ([]domain.RegistryDocument, error) {
	var documents []domain.RegistryDocument

	for _, set := range sets {
		kind := domain.ParseDocumentKind(set.DocName)

		for _, item := range set.List {
			document := domain.RegistryDocument{
				Kind:               kind,
				RegistrationNumber: strings.TrimSpace(item.RegNumber),
				Link:               strings.TrimSpace(item.BlobURI),
			}

			if text := strings.TrimSpace(item.DocDate); text != "" {
				parsed, err := time.Parse(documentDate, text)
				if err != nil {
					return nil, fmt.Errorf("reading the date of a document: %w", err)
				}

				date, err := kernel.ParseDate(parsed.Format(time.DateOnly))
				if err != nil {
					return nil, fmt.Errorf("reading the date of a document: %w", err)
				}

				document.Date = &date
			}

			documents = append(documents, document)
		}
	}

	return documents, nil
}

func toNearby(item naprclient.NearbyParcel) (domain.NearbyParcel, error) {
	code, err := domain.ParseCadastralCode(item.Name)
	if err != nil {
		return domain.NearbyParcel{}, fmt.Errorf("reading a parcel near a point: %w", err)
	}

	distance, err := strconv.ParseFloat(strings.TrimSpace(item.Distance), 64)
	if err != nil || distance < 0 || !finite(distance) {
		return domain.NearbyParcel{}, fmt.Errorf("%w: %q", errBadDistance, item.Distance)
	}

	return domain.NearbyParcel{
		Code:      code,
		Address:   domain.CollapseSpaces(item.Descript),
		DistanceM: int(math.Round(distance)),
	}, nil
}

func shapeCentre(shapes []naprclient.Shape) (domain.Point, error) {
	if len(shapes) == 0 {
		return domain.Point{}, errNoShape
	}

	middle, err := centre(shapes[0].Shape)
	if err != nil {
		return domain.Point{}, err
	}

	point, err := domain.NewPoint(middle.y, middle.x)
	if err != nil {
		return domain.Point{}, fmt.Errorf("reading the centre of a shape: %w", err)
	}

	return point, nil
}

func unavailable(err error) error {
	return fmt.Errorf("reading NAPR: %w: %w", kernel.ErrSourceUnavailable, err)
}
