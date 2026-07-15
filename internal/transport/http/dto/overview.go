package dto

import "github.com/YuriyDubinin/bms-api/internal/service"

// DealOverviewHTTPResponse — сделка + связанные сущности одним ответом.
type DealOverviewHTTPResponse struct {
	Deal    DealHTTPResponse     `json:"deal"`
	Project *ProjectHTTPResponse `json:"project"`
	Client  *ClientHTTPResponse  `json:"client"`
	Service *ServiceHTTPResponse `json:"service"`
	Tasks   []TaskHTTPResponse   `json:"tasks"`
	Links   []LinkHTTPResponse   `json:"links"`
}

func FromDealOverview(o *service.DealOverviewOutput) DealOverviewHTTPResponse {
	resp := DealOverviewHTTPResponse{
		Deal:  fromDealItem(o.Deal),
		Tasks: make([]TaskHTTPResponse, 0, len(o.Tasks)),
		Links: make([]LinkHTTPResponse, 0, len(o.Links)),
	}
	if o.Project != nil {
		p := fromProjectItem(*o.Project)
		resp.Project = &p
	}
	if o.Client != nil {
		c := fromClientItem(*o.Client)
		resp.Client = &c
	}
	if o.Service != nil {
		s := fromServiceItem(*o.Service)
		resp.Service = &s
	}
	for _, t := range o.Tasks {
		resp.Tasks = append(resp.Tasks, fromTaskItem(t))
	}
	for _, l := range o.Links {
		resp.Links = append(resp.Links, fromLinkItem(l))
	}
	return resp
}

// ClientOverviewHTTPResponse — клиент + связанные сущности одним ответом.
type ClientOverviewHTTPResponse struct {
	Client  ClientHTTPResponse   `json:"client"`
	Project *ProjectHTTPResponse `json:"project"`
	Deals   []DealHTTPResponse   `json:"deals"`
	Tasks   []TaskHTTPResponse   `json:"tasks"`
	Links   []LinkHTTPResponse   `json:"links"`
}

func FromClientOverview(o *service.ClientOverviewOutput) ClientOverviewHTTPResponse {
	resp := ClientOverviewHTTPResponse{
		Client: fromClientItem(o.Client),
		Deals:  make([]DealHTTPResponse, 0, len(o.Deals)),
		Tasks:  make([]TaskHTTPResponse, 0, len(o.Tasks)),
		Links:  make([]LinkHTTPResponse, 0, len(o.Links)),
	}
	if o.Project != nil {
		p := fromProjectItem(*o.Project)
		resp.Project = &p
	}
	for _, d := range o.Deals {
		resp.Deals = append(resp.Deals, fromDealItem(d))
	}
	for _, t := range o.Tasks {
		resp.Tasks = append(resp.Tasks, fromTaskItem(t))
	}
	for _, l := range o.Links {
		resp.Links = append(resp.Links, fromLinkItem(l))
	}
	return resp
}
