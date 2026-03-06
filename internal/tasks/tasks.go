package tasks

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Task represents a TODO/FIXME extracted from markdown files.
type Task struct {
	File     string
	Line     int
	Text     string
	Checked  bool
	Priority string   // "high", "medium", "low", or ""
	Tags     []string // extracted #tag values
	DueDate  string   // extracted @due(YYYY-MM-DD) value
}

var taskPattern = regexp.MustCompile(`^(\s*[-*]\s+\[([xX ])\]\s+)(.+)$`)
var priorityPattern = regexp.MustCompile(`!(\w+)\s*`)
var tagPattern = regexp.MustCompile(`#(\w[\w-]*)`)
var dueDatePattern = regexp.MustCompile(`@due\(([^)]+)\)`)

// Extract finds all task items in markdown content.
func Extract(filePath, content string) []Task {
	var tasks []Task
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		matches := taskPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		checked := matches[2] == "x" || matches[2] == "X"
		text := strings.TrimSpace(matches[3])

		t := Task{
			File:    filePath,
			Line:    i + 1,
			Text:    text,
			Checked: checked,
		}

		// Extract priority: !high, !medium, !low
		if pm := priorityPattern.FindStringSubmatch(text); pm != nil {
			p := strings.ToLower(pm[1])
			if p == "high" || p == "medium" || p == "low" {
				t.Priority = p
				t.Text = strings.TrimSpace(priorityPattern.ReplaceAllString(t.Text, ""))
			}
		}

		// Extract tags: #tag1 #tag2
		if tm := tagPattern.FindAllStringSubmatch(text, -1); tm != nil {
			for _, m := range tm {
				t.Tags = append(t.Tags, m[1])
			}
			t.Text = strings.TrimSpace(tagPattern.ReplaceAllString(t.Text, ""))
		}

		// Extract due date: @due(2024-01-15)
		if dm := dueDatePattern.FindStringSubmatch(text); dm != nil {
			t.DueDate = dm[1]
			t.Text = strings.TrimSpace(dueDatePattern.ReplaceAllString(t.Text, ""))
		}

		tasks = append(tasks, t)
	}

	return tasks
}

// Aggregate collects tasks from multiple files.
func Aggregate(fileContents map[string]string) []Task {
	var all []Task
	for path, content := range fileContents {
		all = append(all, Extract(path, content)...)
	}
	return all
}

// Pending returns only unchecked tasks.
func Pending(tasks []Task) []Task {
	var pending []Task
	for _, t := range tasks {
		if !t.Checked {
			pending = append(pending, t)
		}
	}
	return pending
}

// GroupByFile groups tasks by their source file.
func GroupByFile(tasks []Task) map[string][]Task {
	groups := make(map[string][]Task)
	for _, t := range tasks {
		groups[t.File] = append(groups[t.File], t)
	}
	return groups
}

// GroupByTag groups tasks by their tags. A task with multiple tags appears in each group.
func GroupByTag(tasks []Task) map[string][]Task {
	groups := make(map[string][]Task)
	for _, t := range tasks {
		if len(t.Tags) == 0 {
			groups["untagged"] = append(groups["untagged"], t)
			continue
		}
		for _, tag := range t.Tags {
			groups[tag] = append(groups[tag], t)
		}
	}
	return groups
}

// FormatTable returns a formatted string table of tasks.
func FormatTable(tasks []Task) string {
	if len(tasks) == 0 {
		return "No tasks found."
	}

	// Sort: unchecked first, then by priority (high > medium > low > ""), then by file/line
	sorted := make([]Task, len(tasks))
	copy(sorted, tasks)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Checked != sorted[j].Checked {
			return !sorted[i].Checked
		}
		pi := priorityRank(sorted[i].Priority)
		pj := priorityRank(sorted[j].Priority)
		if pi != pj {
			return pi < pj
		}
		if sorted[i].File != sorted[j].File {
			return sorted[i].File < sorted[j].File
		}
		return sorted[i].Line < sorted[j].Line
	})

	// Calculate column widths
	maxText := 4 // "Text"
	maxFile := 4 // "File"
	maxPri := 8  // "Priority"
	maxDue := 3  // "Due"
	for _, t := range sorted {
		if len(t.Text) > maxText {
			maxText = len(t.Text)
		}
		loc := fmt.Sprintf("%s:%d", t.File, t.Line)
		if len(loc) > maxFile {
			maxFile = len(loc)
		}
		if len(t.Priority) > maxPri {
			maxPri = len(t.Priority)
		}
		if len(t.DueDate) > maxDue {
			maxDue = len(t.DueDate)
		}
	}

	// Cap text width
	if maxText > 60 {
		maxText = 60
	}

	var b strings.Builder

	// Header
	fmt.Fprintf(&b, "%-3s %-*s %-*s %-*s %-*s %s\n",
		"[?]", maxText, "Text", maxFile, "File", maxPri, "Priority", maxDue, "Due", "Tags")
	b.WriteString(strings.Repeat("-", 3+1+maxText+1+maxFile+1+maxPri+1+maxDue+1+10) + "\n")

	for _, t := range sorted {
		check := "[ ]"
		if t.Checked {
			check = "[x]"
		}

		text := t.Text
		if len(text) > maxText {
			text = text[:maxText-3] + "..."
		}

		loc := fmt.Sprintf("%s:%d", t.File, t.Line)
		tags := strings.Join(t.Tags, ", ")

		fmt.Fprintf(&b, "%s %-*s %-*s %-*s %-*s %s\n",
			check, maxText, text, maxFile, loc, maxPri, t.Priority, maxDue, t.DueDate, tags)
	}

	fmt.Fprintf(&b, "\nTotal: %d tasks (%d pending)\n", len(sorted), len(Pending(sorted)))

	return b.String()
}

func priorityRank(p string) int {
	switch p {
	case "high":
		return 0
	case "medium":
		return 1
	case "low":
		return 2
	default:
		return 3
	}
}
