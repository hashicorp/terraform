package models

type RouteSummary struct {
	GUID   string
	Host   string
	Domain DomainFields
	Path   string
	Port   int
}

func (r RouteSummary) URL() string {
	return (&RoutePresenter{
		Host:   r.Host,
		Domain: r.Domain.Name,
		Path:   r.Path,
		Port:   r.Port,
	}).URL()
}
