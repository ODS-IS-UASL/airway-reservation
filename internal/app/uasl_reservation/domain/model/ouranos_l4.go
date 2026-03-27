package model

type UaslServiceURL struct {
	DiscoveryServiceName string
	DomainAppID          string
	BaseURL              string
}

type FindResourceRequest struct {
	ServiceName string
	Lat         float64
	Lng         float64
	RadiusKm    float64
}
