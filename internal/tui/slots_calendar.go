package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/joaodiniz/42cli/internal/models"
)

const (
	calendarDays  = 7
	calendarStart = 8  // inclusive hour
	calendarEnd   = 22 // inclusive hour
)

var weekdayShort = [...]string{"dom", "seg", "ter", "qua", "qui", "sex", "sáb"}

// slotCell is the state of one hour on the calendar grid.
type slotCell int

const (
	cellEmpty slotCell = iota
	cellFree
	cellBooked
)

// RenderSlotsCalendar draws a 7-day × hour grid of the user's slots.
// now anchors "today"; slots without begin/end are ignored.
func RenderSlotsCalendar(slots []models.Slot, slotsErr string, now time.Time) string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Meus slots"))

	if slotsErr != "" {
		b.WriteString("\n")
		b.WriteString(styleFail.Render("indisponível: " + slotsErr))
		b.WriteString("\n")
		b.WriteString(styleLabel.Render("Ative o scope projects e rode `lightyear logout && lightyear login`."))
		return styleCard.Render(b.String())
	}

	if len(slots) == 0 {
		b.WriteString("\n")
		b.WriteString(styleLabel.Render("Nenhum slot aberto. `lightyear slots open --duration 1h`"))
		return styleCard.Render(b.String())
	}

	grid := buildSlotCalendar(slots, now)
	b.WriteString("\n")
	b.WriteString(renderSlotCalendarHeader())
	for i := 0; i < calendarDays; i++ {
		day := startOfLocalDay(now).AddDate(0, 0, i)
		b.WriteString("\n")
		b.WriteString(renderSlotCalendarRow(day, grid[dayKey(day)]))
	}
	b.WriteString("\n")
	b.WriteString(styleLabel.Render("██ livre  ▓▓ agendado  ·· vazio"))
	return styleCard.Render(b.String())
}

// buildSlotCalendar maps day → hour → cell for the next calendarDays.
func buildSlotCalendar(slots []models.Slot, now time.Time) map[string]map[int]slotCell {
	start := startOfLocalDay(now)
	end := start.AddDate(0, 0, calendarDays)

	grid := make(map[string]map[int]slotCell, calendarDays)
	for i := 0; i < calendarDays; i++ {
		day := start.AddDate(0, 0, i)
		hours := make(map[int]slotCell, calendarEnd-calendarStart+1)
		for h := calendarStart; h <= calendarEnd; h++ {
			hours[h] = cellEmpty
		}
		grid[dayKey(day)] = hours
	}

	for _, slot := range slots {
		if slot.BeginAt == nil || slot.EndAt == nil {
			continue
		}
		begin := slot.BeginAt.Local()
		endAt := slot.EndAt.Local()
		if !endAt.After(start) || !begin.Before(end) {
			continue
		}

		next := cellFree
		if slot.Booked() {
			next = cellBooked
		}

		for cursor := begin.Truncate(time.Hour); cursor.Before(endAt); cursor = cursor.Add(time.Hour) {
			day := startOfLocalDay(cursor)
			if day.Before(start) || !day.Before(end) {
				continue
			}
			hour := cursor.Hour()
			if hour < calendarStart || hour > calendarEnd {
				continue
			}
			key := dayKey(day)
			cur := grid[key][hour]
			if cur == cellEmpty || next == cellBooked {
				grid[key][hour] = next
			}
		}
	}
	return grid
}

func renderSlotCalendarHeader() string {
	var b strings.Builder
	b.WriteString(styleLabel.Render(fmt.Sprintf("%-10s", "")))
	for h := calendarStart; h <= calendarEnd; h++ {
		b.WriteString(styleLabel.Render(fmt.Sprintf(" %02d", h)))
	}
	return b.String()
}

func renderSlotCalendarRow(day time.Time, hours map[int]slotCell) string {
	label := fmt.Sprintf("%s %s", weekdayShort[day.Weekday()], day.Format("02/01"))
	var b strings.Builder
	b.WriteString(styleValue.Render(fmt.Sprintf("%-10s", label)))
	for h := calendarStart; h <= calendarEnd; h++ {
		b.WriteString(" ")
		b.WriteString(renderSlotCell(hours[h]))
	}
	return b.String()
}

func renderSlotCell(cell slotCell) string {
	switch cell {
	case cellFree:
		return styleGood.Render("██")
	case cellBooked:
		return styleAccent.Render("▓▓")
	default:
		return styleLabel.Render("··")
	}
}

func startOfLocalDay(t time.Time) time.Time {
	t = t.Local()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func dayKey(t time.Time) string {
	return t.Format("2006-01-02")
}
