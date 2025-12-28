package stacksview

import (
        "fmt"
        "strings"
        "swarmcli/docker"
        "swarmcli/ui"
        filterlist "swarmcli/ui/components/filterable/list"

        "github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
        if !m.Visible {
                return ""
        }

        title := fmt.Sprintf("Stacks on Node (Total: %d)", len(m.List.Items))

        // Compute five percentage-based column widths so columns start at
        // 0%, 20%, 40%, 60%, 80% of the available content width.
        width := m.List.Viewport.Width
        if width <= 0 {
                width = m.width
        }
        if width <= 0 {
                width = 80
        }
        contentWidth := width
        base := contentWidth / 5
        colWidths := make([]int, 5)
        for i := 0; i < 5; i++ {
                colWidths[i] = base
        }
        rem := contentWidth - base*5
        for i := 0; i < rem && i < 5; i++ {
                colWidths[i]++
        }

        // Build header using frame header style so it appears on the first
        // line inside the framed box and aligns with rows below.
        headerLine := fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s",
                colWidths[0], "  STACK",
                colWidths[1], "SERVICES",
                colWidths[2], "NODES",
                colWidths[3], "",
                colWidths[4], "",
        )
        header := ui.FrameHeaderStyle.Render(headerLine)

        // Footer: cursor + optional search query
        status := fmt.Sprintf("Stack %d of %d", m.List.Cursor+1, len(m.List.Filtered))
        statusBar := ui.StatusBarStyle.Render(status)

        var footer string
        if m.List.Mode == filterlist.ModeSearching {
                footer = ui.StatusBarStyle.Render("Filter (type then Enter): " + m.List.Query)
        } else if m.List.Query != "" {
                footer = ui.StatusBarStyle.Render("Filter: " + m.List.Query)
        }

        if footer != "" {
                footer = statusBar + "\n" + footer
        } else {
                footer = statusBar
        }

        // Set RenderItem to format rows using the same colWidths so the
        // header and rows align exactly.
        m.List.RenderItem = func(s docker.StackEntry, selected bool, _ int) string {
                // First column: current marker + name (we don't have a marker here but keep spacing)
                nameMax := colWidths[0] - 2
                if nameMax < 0 {
                        nameMax = 0
                }
                name := s.Name
                if len(name) > nameMax {
                        if nameMax > 3 {
                                name = name[:nameMax-3] + "..."
                        } else {
                                name = name[:nameMax]
                        }
                }
                first := fmt.Sprintf("  %s", name)

                svcStr := fmt.Sprintf("%d", s.ServiceCount)
                svcMax := colWidths[1]
                if len(svcStr) > svcMax {
                        svcStr = svcStr[:svcMax]
                }

                nodeStr := fmt.Sprintf("%d", s.NodeCount)
                nodeMax := colWidths[2]
                if len(nodeStr) > nodeMax {
                        nodeStr = nodeStr[:nodeMax]
                }

                // Empty placeholders for remaining columns
                col4 := ""
                col5 := ""

                line := fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s",
                        colWidths[0], first,
                        colWidths[1], svcStr,
                        colWidths[2], nodeStr,
                        colWidths[3], col4,
                        colWidths[4], col5,
                )
                if selected {
                        selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("63")).
Bold(true)                                                                                                                                         return selStyle.Render(line)
                }
                return lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Render(line)
        }

        // Content rendered by the FilterableList
        content := m.List.View()

        // Add 4 to make frame full terminal width (app reduces viewport by 4 in normal mode)
        frameWidth := m.List.Viewport.Width
        if frameWidth <= 0 {
                // Fallback to model width if viewport hasn't been initialized yet
                frameWidth = m.width
        }
        frameWidth = frameWidth + 4

        // Compute frameHeight from viewport (treat Viewport.Height as the total
        // frame height like `configs` view does). Then compute desired inner
        // content lines = frameHeight - borders - header - footer, and pad/trim
        // content to that length.
        // Use the adjusted viewport height directly; the framing helper
        // will account for borders. Do not subtract extra rows here.
        // Reserve two lines from the viewport height for surrounding UI (helpbar/systeminfo)
        frameHeight := m.List.Viewport.Height - 2
        if frameHeight <= 0 {
                // Fallback to model height minus reserved lines if viewport not initialized
                if m.height > 0 {
                        frameHeight = m.height - 4
                }
                if frameHeight <= 0 {
                        frameHeight = 20
                }
        }

        // Header occupies one line when present (styled header renders single line)
        headerLines := 0
        if header != "" {
                headerLines = 1
        }
        footerLines := 0
        if footer != "" {
                footerLines = len(strings.Split(footer, "\n"))
        }

        desiredContentLines := frameHeight - 2 - headerLines - footerLines
        if desiredContentLines < 0 {
                desiredContentLines = 0
        }

        contentLines := strings.Split(content, "\n")
        // Trim trailing empty lines
        for len(contentLines) > 0 && contentLines[len(contentLines)-1] == "" {
                contentLines = contentLines[:len(contentLines)-1]
        }
        if len(contentLines) < desiredContentLines {
                for i := 0; i < desiredContentLines-len(contentLines); i++ {
                        contentLines = append(contentLines, "")
                }
        } else if len(contentLines) > desiredContentLines {
                contentLines = contentLines[:desiredContentLines]
        }
        paddedContent := strings.Join(contentLines, "\n")

        framed := ui.RenderFramedBoxHeight(title, header, paddedContent, footer, frameWidth, frameHeight)

        return framed
}
