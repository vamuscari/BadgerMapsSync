package test

import (
	"badgermaps/app"
	"badgermaps/app/test"
)

// CliPresenter handles the presentation logic for the test command.
type CliPresenter struct {
	App *app.App
}

// NewCliPresenter creates a new presenter for the test command.
func NewCliPresenter(a *app.App) *CliPresenter {
	return &CliPresenter{App: a}
}

// HandleRunTests runs all tests.
func (p *CliPresenter) HandleRunTests() {
	test.RunTests(p.App)
}

// HandleTestDatabase tests the database.
func (p *CliPresenter) HandleTestDatabase() {
	test.TestDatabase(p.App)
}

// HandleTestApi tests the API.
func (p *CliPresenter) HandleTestApi(save bool) {
	test.TestApi(p.App, save)
}
