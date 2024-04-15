package models

import "time"

type BookmarkEntry struct {
	URL string `json:"url"`
}

type BookmarkList struct {
	BookmarkEntry []BookmarkEntry `json:"bookmarks"`
}

type BookmarksResponse struct {
	TotalCount   int             `json:"totalCount"`
	Next         string          `json:"next"`
	BookmarkList []BookmarkEntry `json:"bookmarks"`
}

type DistributeBookmarksRequest struct {
	DeviceIDs []int `json:"devicesIds"`
}

type InvalidDistributeBookmarksResponse struct {
	Error     string `json:"error"`
	DeviceIDs []int  `json:"invalidDevicesIds"`
}

type InValidBookmarksResponse struct {
	Error        string          `json:"error"`
	BookmarkList []BookmarkEntry `json:"invalidBookmarks"`
}

type BookmarksConfig struct {
	Enabled bool `json:"enabled"`
}

type WebCrawlerJob struct {
	ID             string    `json:"id"`
	DeviceId       int       `json:"deviceId"`
	PackageVersion string    `json:"packageVersion"`
	State          string    `json:"state"`
	StatusMessage  string    `json:"statusMessage"`
	StartTime      time.Time `json:"started"`
	EndTime        time.Time `json:"ended"`
}

type DistributedBookmarksResponse struct {
	DistributionJobList []WebCrawlerJob `json:"distributionJobs"`
	TotalCount          int             `json:"totalCount"`
	Next                string          `json:"next"`
}
