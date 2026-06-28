package ui

import "github.com/charmbracelet/bubbles/list"

func newBrowseList(title string, items []list.Item, width, height int) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	browseList := list.New(items, delegate, 0, 0)
	browseList.Title = title
	browseList.SetShowFilter(true)
	browseList.SetFilteringEnabled(true)
	browseList.Styles.Title = titleStyle
	browseList.Styles.PaginationStyle = paginationStyle
	browseList.Styles.HelpStyle = helpStyle
	resizeBrowseList(&browseList, width, height)
	return browseList
}

func resizeBrowseList(browseList *list.Model, width, height int) {
	if width > 0 && height > 0 {
		browseList.SetSize(width/2-4, height-4)
	}
}

func resizeBrowseListForFetch(browseList *list.Model, width, height int) {
	const footerHeight = 3
	availableHeight := height - footerHeight - 5
	browseList.SetSize(width/2-4, availableHeight)
}

func replaceBrowseListItems(browseList *list.Model, items []list.Item) {
	filterState := browseList.FilterState()
	filterValue := browseList.FilterValue()

	browseList.SetItems(items)
	browseList.ResetSelected()

	if filterState == list.Filtering {
		browseList.ResetFilter()
		browseList.FilterInput.SetValue(filterValue)
	}
}
