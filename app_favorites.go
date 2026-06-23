package main

import (
	"fmt"
	"inklab/backend/database"
)

// AddFavorite adds an item to favorites with optional category
func (a *App) AddFavorite(itemEntry int, category string) *database.FavoriteResult {
	fmt.Printf("[API] AddFavorite called: item=%d, category='%s'\n", itemEntry, category)

	err := a.favoriteRepo.AddFavorite(itemEntry, category)
	if err != nil {
		fmt.Printf("[API] AddFavorite error: %v\n", err)
		return &database.FavoriteResult{
			Success: false,
			Message: err.Error(),
		}
	}

	return &database.FavoriteResult{
		Success: true,
		Message: "Item added to favorites",
	}
}

// RemoveFavorite removes an item from favorites
func (a *App) RemoveFavorite(itemEntry int) *database.FavoriteResult {
	fmt.Printf("[API] RemoveFavorite called: item=%d\n", itemEntry)

	err := a.favoriteRepo.RemoveFavorite(itemEntry)
	if err != nil {
		return &database.FavoriteResult{
			Success: false,
			Message: err.Error(),
		}
	}

	return &database.FavoriteResult{
		Success: true,
		Message: "Item removed from favorites",
	}
}

// IsFavorite checks if an item is in favorites
func (a *App) IsFavorite(itemEntry int) bool {
	isFav, err := a.favoriteRepo.IsFavorite(itemEntry)
	if err != nil {
		fmt.Printf("[API] IsFavorite error: %v\n", err)
		return false
	}
	return isFav
}

// GetAllFavorites returns all favorite items
func (a *App) GetAllFavorites() []*database.FavoriteItem {
	fmt.Println("[API] GetAllFavorites called")
	favorites, err := a.favoriteRepo.GetAllFavorites()
	if err != nil {
		fmt.Printf("[API] GetAllFavorites error: %v\n", err)
		return []*database.FavoriteItem{}
	}
	fmt.Printf("[API] GetAllFavorites returning %d items\n", len(favorites))
	return favorites
}

// GetFavoritesByCategory returns favorites filtered by category
func (a *App) GetFavoritesByCategory(category string) []*database.FavoriteItem {
	favorites, err := a.favoriteRepo.GetFavoritesByCategory(category)
	if err != nil {
		fmt.Printf("[API] GetFavoritesByCategory error: %v\n", err)
		return []*database.FavoriteItem{}
	}
	return favorites
}

// GetFavoriteCategories returns all distinct favorite categories
func (a *App) GetFavoriteCategories() []*database.FavoriteCategory {
	cats, err := a.favoriteRepo.GetCategories()
	if err != nil {
		fmt.Printf("[API] GetFavoriteCategories error: %v\n", err)
		return []*database.FavoriteCategory{}
	}
	return cats
}

// UpdateFavoriteCategory updates the category of a favorite
func (a *App) UpdateFavoriteCategory(itemEntry int, category string) *database.FavoriteResult {
	fmt.Printf("[API] UpdateFavoriteCategory: item=%d, category='%s'\n", itemEntry, category)

	err := a.favoriteRepo.UpdateCategory(itemEntry, category)
	if err != nil {
		return &database.FavoriteResult{
			Success: false,
			Message: err.Error(),
		}
	}

	return &database.FavoriteResult{
		Success: true,
		Message: "Category updated",
	}
}

// UpdateFavoriteStatus updates the status of a favorite (0=None, 1=Obtained, 2=Abandoned)
func (a *App) UpdateFavoriteStatus(itemEntry int, status int) *database.FavoriteResult {
	fmt.Printf("[API] UpdateFavoriteStatus: item=%d, status=%d\n", itemEntry, status)

	err := a.favoriteRepo.UpdateStatus(itemEntry, status)
	if err != nil {
		return &database.FavoriteResult{
			Success: false,
			Message: err.Error(),
		}
	}

	return &database.FavoriteResult{
		Success: true,
		Message: "Status updated",
	}
}

// ToggleFavorite toggles favorite status and returns current state
func (a *App) ToggleFavorite(itemEntry int, category string) *database.FavoriteResult {
	isFav, _ := a.favoriteRepo.IsFavorite(itemEntry)

	if isFav {
		return a.RemoveFavorite(itemEntry)
	}
	return a.AddFavorite(itemEntry, category)
}
