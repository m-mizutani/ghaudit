package model

import (
	"github.com/google/go-github/v42/github"
)

type RegoInput struct {
	Repo          *github.Repository `json:"repo"`
	Branches      []*github.Branch   `json:"branches"`
	Collaborators []*github.User     `json:"collaborators"`
	Hooks         []*github.Hook     `json:"hooks"`
	Teams         []*github.Team     `json:"teams"`
	Timestamp     int64              `json:"timestamp"`
}

type RegoOutput struct {
	Fail []*RegoFail `json:"fail"`
}

type RegoFail struct {
	Category string `json:"category"`
	Message  string `json:"message"`
}
