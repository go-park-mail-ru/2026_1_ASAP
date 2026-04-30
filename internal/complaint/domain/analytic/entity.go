package analytic

type ComplaintAnalytic struct {
	CountType   CountType
	CountStatus CountStatus
}

type CountType struct {
	CountBug     int64
	CountUpgrade int64
	CountProduct int64
}

type CountStatus struct {
	CountStatusOpened int64
	CountStatusInWork int64
	CountStatusClosed int64
}
