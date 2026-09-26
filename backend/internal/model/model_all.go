package model

import "gorm.io/gorm"

func AllModels() []any {
	return []any{&Category{}, &Product{}, &Supplier{}, &Offer{}, &PriceHistory{}, &Favorite{}, &PriceAlert{}, &Budget{}, &PurchaseOrder{}, &Arrival{}}
}
func Migrate(db *gorm.DB) error { return db.AutoMigrate(AllModels()...) }
