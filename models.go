// models.go

package main

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
    ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Name     string             `json:"name"`
    Email    string             `json:"email"`
    Phone    string             `json:"phone"`
    Password string             `json:"password,omitempty"`
}

type Product struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Name        string             `json:"name"`
    Price       int                `json:"price"`
    Image       string             `json:"image"`
    Description string             `json:"description"`
    Stock	int		   `json:"stock"`
}

type CartItem struct {
    ProductID primitive.ObjectID `bson:"productId" json:"productId"`
    Quantity  int                `json:"quantity"`
}

type Cart struct {
    ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID primitive.ObjectID `json:"userId"`
    Items  []CartItem         `json:"items"`
}

type Wishlist struct {
    ID      primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
    UserID  primitive.ObjectID   `json:"userId"`
    ProductIDs []primitive.ObjectID `json:"productIds"`
}

type Order struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID    primitive.ObjectID `bson:"userId" json:"userId"`
    Address   string             `json:"address"`
    Items     []CartItem         `json:"items"`
    Total     int                `json:"total"`
    Date      string             `json:"date"`
    Status    string             `json:"status"`
}
