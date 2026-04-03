package server

type SyncJobSubmitRequest struct {
	Mode       string `json:"mode"`
	Source     string `json:"source,omitempty"`
	Name       string `json:"name,omitempty"`
	ResourceID int    `json:"resource_id,omitempty"`
}

type SyncJobListResponse struct {
	Activity RuntimeActivity `json:"activity"`
	Jobs     []*SyncJob      `json:"jobs"`
}
