# TODO

- [x] Respect configured concurrency limits by copying `Config.MaxConcurrentRequests` into `App.MaxConcurrentRequests` before clamping; right now `LoadConfig` leaves `MaxConcurrentRequests` at zero so pulls always run with the hardcoded default and ignore user overrides (`app/app.go:189`).
- [x] Update `ExecAction` to support structured `Args` when `use_shell` is set to false while keeping shell execution the default for backwards compatibility (`app/action/actions.go`).
- [x] Handle errors from `json.Unmarshal` when decoding pending change payloads so we don't push malformed data or panic on nil maps (see `app/push/actions_push.go:34` and `app/push/actions_push.go:83`).
- [x] Surface database connection failures during `App.LoadConfig` by dispatching/logging the error and clearing `App.DB` when `Connect` fails instead of silently continuing with a disconnected handle (`app/app.go:165`).
- [x] GUI Home: Make the "Refresh" button wider so it matches the primary control sizing expectations (`temp_task_list.md`).
- [x] GUI Sync Center: Create a History table in the schema to track sync runs for later display (`temp_task_list.md`).
- [x] GUI Sync Center: Emit and capture events to populate the History view with actionable data (`temp_task_list.md`).
- [ ] GUI Sync Center: Relocate sync settings to the server/config sections so tabs can be removed (`temp_task_list.md`).
- [ ] GUI Sync Center: Remove the remaining tabs and deliver a streamlined single-pane layout after settings relocate (`temp_task_list.md`).
- [ ] GUI Sync Center: Research and approve an improved layout that supports history and settings changes (`temp_task_list.md`).
- [ ] GUI Explorer: Stop column data from bleeding into neighboring columns when widths shrink (`temp_task_list.md`).
- [x] GUI Explorer: Render `\<nil\>` values as blank cells for clarity (`temp_task_list.md`).
- [ ] GUI Explorer: Expand the search bar further left so it has room for multi-word queries (`temp_task_list.md`).
- [ ] GUI Actions: Format event names (e.g., display `pull.group.complete` as "Pull Group Complete") for readability (`temp_task_list.md`).
- [ ] GUI Log: Fix wrapped log lines so they do not overlap subsequent entries (`temp_task_list.md`).
- [x] GUI Layout: Allow clicking the slide-over backdrop to dismiss the panel for faster interaction (`gui/gui.go`).
- [x] GUI Layout: Add a floating button at the top-right of the tab area to toggle the slide-over on demand (`temp_task_list.md`).
