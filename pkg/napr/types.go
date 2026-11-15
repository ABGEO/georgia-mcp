package napr

// SearchResult is one result of a search: a parcel or an address.
type SearchResult struct {
	// Name is the cadastral code of a parcel, or the text of an address.
	Name string `json:"name"`
	// Descript is the address of a parcel, or the municipality of an address.
	Descript string        `json:"descript"`
	Details  SearchDetails `json:"details"`
}

// SearchDetails holds the links of a search result.
type SearchDetails struct {
	// Type is "address" for a parcel and for an address.
	Type string `json:"type"`
	// InfoLink is "/lr/bo/mg/getinfo.alpha?lbl=lr_parcels:..." for a parcel and
	// "/ar/mg/getinfo?lbl=ract_g_borders:..." for an address.
	InfoLink     string `json:"info_link"`
	GeometryLink string `json:"geometry_link"`
}

// NearbyParcel is one parcel near a point.
type NearbyParcel struct {
	Name       string `json:"name"`
	Descript   string `json:"descript"`
	ResultLink string `json:"resultlink"`
	// Distance is in metres: "91.775768836483".
	Distance string `json:"distance"`
}

// ParcelInfo is the detail of a parcel.
type ParcelInfo struct {
	ObjTypeID string        `json:"_obj_type_id_"`
	ObjName   string        `json:"_obj_name_"`
	Detail    ParcelDetail  `json:"_obj_detail_"`
	Owners    []Owner       `json:"_owners_"`
	Docs      []DocumentSet `json:"_docs_"`
}

// ParcelDetail holds the values of a parcel, in Georgian.
type ParcelDetail struct {
	CadCode string `json:"_cadcode_"`
	// AreaMeasured is a decimal text: "618".
	AreaMeasured string `json:"_area_measured_"`
	// Unit is "კვ.მ." for square metres.
	Unit        string `json:"_unit_"`
	Address     string `json:"_address_"`
	LandType    string `json:"_land_type_"`
	OwnTypeName string `json:"_owntype_name_"`
}

// Owner is one owner of a parcel.
type Owner struct {
	Person string `json:"_person_"`
}

// DocumentSet is the documents of one kind.
type DocumentSet struct {
	DocName string     `json:"_doc_name_"`
	List    []Document `json:"_list_"`
}

// Document is one document of the registry.
type Document struct {
	BlobURI   string `json:"_blob_uri_"`
	RegNumber string `json:"_regnumber_"`
	// DocDate is "dd.mm.yyyy", or empty.
	DocDate string `json:"_doc_date_"`
}

// Shape is the shape of a parcel or an address.
type Shape struct {
	Name string `json:"name"`
	// Proj is "EPSG:4326".
	Proj string `json:"proj"`
	// Shape is a WKT POLYGON or MULTIPOLYGON, in "lon lat" order.
	Shape       string `json:"shape"`
	ShapeFormat string `json:"shape_format"`
}
