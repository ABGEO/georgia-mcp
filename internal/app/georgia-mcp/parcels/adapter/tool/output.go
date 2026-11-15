package tool

import (
	"math"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const pointScale = 1e6

// codeNames gives the Georgian name of each code. An unknown code takes the text of NAPR.
func codeNames() map[string]string {
	return map[string]string{
		string(domain.LandAgricultural):    "სასოფლო-სამეურნეო",
		string(domain.LandNonAgricultural): "არასასოფლო-სამეურნეო",
		string(domain.Ownership):           "საკუთრება",
		string(domain.CoOwnership):         "თანასაკუთრება",
		string(domain.RegistryExtract):     "ამონაწერი საჯარო რეესტრიდან",
		string(domain.CadastralPlan):       "საკადასტრო გეგმა",
	}
}

func toCode[T ~string](classified domain.Classified[T], names map[string]string) codeOutput {
	name, ok := names[string(classified.Code)]
	if !ok {
		name = classified.Text
	}

	return codeOutput{Code: string(classified.Code), Name: name}
}

func toParcelOutput(read app.ParcelRead) parcelOutput {
	names := codeNames()
	parcel := read.Parcel

	documents := make([]documentOutput, 0, len(parcel.Documents))
	for _, document := range parcel.Documents {
		output := documentOutput{
			Kind:               toCode(document.Kind, names),
			RegistrationNumber: document.RegistrationNumber,
			URL:                document.Link,
		}
		if document.Date != nil {
			output.Date = document.Date.String()
		}

		documents = append(documents, output)
	}

	owners := parcel.Owners
	if owners == nil {
		owners = []string{}
	}

	return parcelOutput{
		Source:        source,
		AsOf:          instant(read.ReadAt),
		Stale:         read.Stale,
		Code:          parcel.Code.String(),
		MapURL:        read.MapURL,
		Address:       parcel.Address,
		Area:          parcel.Area.Value,
		AreaUnit:      parcel.Area.Unit,
		LandType:      toCode(parcel.LandType, names),
		OwnershipType: toCode(parcel.Ownership, names),
		Owners:        owners,
		Documents:     documents,
		Centre:        toPoint(parcel.Centre),
	}
}

func toNearbyOutput(read app.NearbyRead) nearbyOutput {
	parcels := make([]nearbyParcelOutput, 0, len(read.Parcels))
	for _, parcel := range read.Parcels {
		parcels = append(parcels, nearbyParcelOutput{
			Code:      parcel.Code.String(),
			Address:   parcel.Address,
			DistanceM: parcel.DistanceM,
		})
	}

	return nearbyOutput{
		Source:  source,
		AsOf:    instant(read.ReadAt),
		Stale:   read.Stale,
		Parcels: parcels,
	}
}

func toAddressesOutput(list app.AddressList) addressesOutput {
	addresses := make([]addressOutput, 0, len(list.Addresses))
	for _, address := range list.Addresses {
		addresses = append(addresses, addressOutput{
			Address:      address.Text,
			Municipality: address.Municipality,
			Centre:       toPoint(address.Centre),
		})
	}

	return addressesOutput{
		Source:    source,
		AsOf:      instant(list.ReadAt),
		Stale:     list.Stale,
		Addresses: addresses,
	}
}

func toPoint(point domain.Point) pointOutput {
	return pointOutput{
		Lat: math.Round(point.Lat*pointScale) / pointScale,
		Lon: math.Round(point.Lon*pointScale) / pointScale,
	}
}

func instant(at time.Time) string {
	return at.In(kernel.TbilisiZone()).Format(time.RFC3339)
}
