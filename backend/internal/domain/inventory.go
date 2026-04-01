package domain

import "time"

// Inventory describes an inventory item together with its derived availability.
type Inventory struct {
	ID             int64     `json:"id" example:"1"`
	SKU            string    `json:"sku" example:"SKU-001"`
	Name           string    `json:"name" example:"Widget A"`
	CustomerName   string    `json:"customerName" example:"Acme Corp"`
	PhysicalStock  int64     `json:"physicalStock" example:"120"`
	ReservedStock  int64     `json:"reservedStock" example:"20"`
	AvailableStock int64     `json:"availableStock" example:"100"`
	CreatedAt      time.Time `json:"createdAt" example:"2026-04-01T10:00:00Z"`
	UpdatedAt      time.Time `json:"updatedAt" example:"2026-04-01T12:00:00Z"`
}

func (inventory *Inventory) RefreshAvailable() {
	inventory.AvailableStock = inventory.PhysicalStock - inventory.ReservedStock
}
