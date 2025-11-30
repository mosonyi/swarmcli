// Package polling provides a generic polling mechanism for views
//
// Example Usage in a View:
//
// 1. Add the poller to your Model:
//
//	import "swarmcli/views/polling"
//
//	type Model struct {
//		List   filterlist.FilterableList[docker.NodeEntry]
//		poller *polling.Poller[docker.NodeEntry]
//		// ... other fields
//	}
//
// 2. Initialize the poller in your New() function:
//
//	func New(width, height int) *Model {
//		m := &Model{
//			List: list,
//			// ... other initialization
//		}
//		
//		// Create poller with load function and message builder
//		m.poller = polling.New(
//			func() ([]docker.NodeEntry, error) {
//				snapshot, err := docker.RefreshSnapshot()
//				if err != nil {
//					return nil, err
//				}
//				return snapshot.ToNodeEntries(), nil
//			},
//			func(entries []docker.NodeEntry) tea.Msg {
//				return Msg{Entries: entries}
//			},
//		)
//		
//		return m
//	}
//
// 3. Start polling in Init():
//
//	func (m *Model) Init() tea.Cmd {
//		return m.poller.TickCmd()
//	}
//
// 4. Handle TickMsg and your data message in Update():
//
//	func (m *Model) Update(msg tea.Msg) tea.Cmd {
//		switch msg := msg.(type) {
//		
//		case Msg:
//			// Update hash when receiving new data
//			m.poller.UpdateHash(msg.Entries)
//			m.SetContent(msg.Entries)
//			// Continue polling
//			return m.poller.TickCmd()
//		
//		case polling.TickMsg:
//			// Check for changes only if view is visible
//			if m.Visible {
//				return m.poller.CheckCmd()
//			}
//			// Continue polling even if not visible
//			return m.poller.TickCmd()
//		
//		// ... other cases
//		}
//	}
//
// 5. Route polling.TickMsg in app/update.go:
//
//	case polling.TickMsg:
//		// Forward to current view
//		cmd := m.currentView.Update(msg)
//		return m, cmd
//
// Benefits:
// - No need to write hash computation for each view
// - Consistent polling behavior across all views
// - Easy to change polling interval per view if needed
// - Reusable for future views
