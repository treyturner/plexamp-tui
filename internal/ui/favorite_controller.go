package ui

import (
	"fmt"

	"plexamp-tui/internal/config"

	"github.com/charmbracelet/bubbles/list"
)

type favoriteStore interface {
	Add(config.FavoriteItem) error
	Remove(itemType, metadataKey string) error
	List() ([]config.FavoriteItem, error)
}

type favoriteController struct {
	favorites    *config.Favorites
	playbackList *list.Model
	store        favoriteStore
}

func (m *model) favoritesController() favoriteController {
	if m.playbackConfig == nil {
		m.playbackConfig = &config.Favorites{}
	}
	var store favoriteStore
	if m.deps.favsManager != nil {
		store = m.deps.favsManager
	}
	return favoriteController{
		favorites:    m.playbackConfig,
		playbackList: &m.playbackList,
		store:        store,
	}
}

func favoriteListItems(favorites []config.FavoriteItem) []list.Item {
	items := make([]list.Item, 0, len(favorites))
	for _, fav := range favorites {
		items = append(items, item{Name: fav.Name, Type: fav.Type, MetadataKey: fav.MetadataKey})
	}
	return items
}

func (c favoriteController) metadataKeySet() map[string]struct{} {
	favSet := make(map[string]struct{})
	if c.favorites == nil {
		return favSet
	}
	for _, fav := range c.favorites.Items {
		if fav.MetadataKey != "" {
			favSet[fav.MetadataKey] = struct{}{}
		}
	}
	return favSet
}

func (c favoriteController) findSelected(selected item) (config.FavoriteItem, bool) {
	if c.favorites == nil {
		return config.FavoriteItem{}, false
	}

	for _, fav := range c.favorites.Items {
		if fav.MetadataKey != "" && fav.MetadataKey == selected.MetadataKey && fav.Type == selected.Type {
			return fav, true
		}
	}
	for _, fav := range c.favorites.Items {
		if fav.Name == string(selected.Name) && fav.Type == selected.Type {
			return fav, true
		}
	}
	return config.FavoriteItem{}, false
}

func (c favoriteController) toggle(name, metadataKey string, itemType favoriteType) (bool, error) {
	if _, _, found := c.findByTypeKey(itemType, metadataKey); found {
		return false, c.remove(itemType, metadataKey)
	}
	return true, c.add(name, metadataKey, itemType)
}

func (c favoriteController) add(name, metadataKey string, itemType favoriteType) error {
	fav := config.FavoriteItem{Name: name, Type: string(itemType), MetadataKey: metadataKey}
	if c.store != nil {
		if err := c.store.Add(fav); err != nil {
			return err
		}
		return c.reloadFromStore()
	}

	if index, _, found := c.findByTypeKey(itemType, metadataKey); found {
		c.favorites.Items[index].Name = fav.Name
		c.favorites.Items[index].Type = fav.Type
		c.favorites.Items[index].MetadataKey = fav.MetadataKey
	} else {
		c.favorites.Items = append(c.favorites.Items, fav)
	}
	c.refreshList()
	return nil
}

func (c favoriteController) remove(itemType favoriteType, metadataKey string) error {
	if c.store != nil {
		if err := c.store.Remove(string(itemType), metadataKey); err != nil {
			return err
		}
		return c.reloadFromStore()
	}

	if index, _, found := c.findByTypeKey(itemType, metadataKey); found {
		c.favorites.Items = append(c.favorites.Items[:index], c.favorites.Items[index+1:]...)
		c.refreshList()
	}
	return nil
}

func (c favoriteController) removeAt(index int) error {
	if c.playbackList == nil || index < 0 || index >= len(c.playbackList.Items()) {
		return fmt.Errorf("favorite index %d out of range", index)
	}
	favToRemove, ok := c.playbackList.Items()[index].(item)
	if !ok {
		return fmt.Errorf("selected favorite has unexpected type %T", c.playbackList.Items()[index])
	}
	return c.remove(favoriteType(favToRemove.Type), favToRemove.MetadataKey)
}

func (c favoriteController) refreshList() {
	if c.playbackList == nil || c.favorites == nil {
		return
	}
	c.playbackList.SetItems(favoriteListItems(c.favorites.Items))
}

func (c favoriteController) reloadFromStore() error {
	if c.store == nil || c.favorites == nil {
		return nil
	}
	items, err := c.store.List()
	if err != nil {
		return err
	}
	c.favorites.Items = items
	c.refreshList()
	return nil
}

func (c favoriteController) findByTypeKey(itemType favoriteType, metadataKey string) (int, config.FavoriteItem, bool) {
	if c.favorites == nil {
		return -1, config.FavoriteItem{}, false
	}
	for index, fav := range c.favorites.Items {
		if fav.Type == string(itemType) && fav.MetadataKey == metadataKey {
			return index, fav, true
		}
	}
	return -1, config.FavoriteItem{}, false
}
