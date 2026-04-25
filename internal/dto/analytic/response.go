package analytic

type ResponseComplaintAnalytic struct {
	CountStatus CountStatus `json:"count_status"`
	CountType   CountType   `json:"count_type"`
}

type CountStatus struct {
	CountStatusOpened int64 `json:"count_status_opened"`
	CountStatusInWork int64 `json:"count_status_in_work"`
	CountStatusClosed int64 `json:"count_status_closed"`
}

type CountType struct {
	CountBug     int64 `json:"count_type_bug"`
	CountUpgrade int64 `json:"count_type_upgrade"`
	CountProduct int64 `json:"count_type_product"`
}
