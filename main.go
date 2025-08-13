// main.go

package main

import (
    "context"
    "time"
    "log"
    "os"
    "fmt"
    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "golang.org/x/crypto/bcrypt"
    "github.com/dgrijalva/jwt-go"
    "github.com/gin-contrib/cors"
)


var (
    dbClient *mongo.Client
    db       *mongo.Database
    jwtSecret = []byte("SECRET")
)

func main() {
    // Connect to MongoDB
    mongoUri := os.Getenv("MONGO_PUBLIC_URL")
    if mongoUri == "" {
    	mongoUri = os.Getenv("MONGO_URL")
    }
    if mongoUri == "" {
        mongoUri = "mongodb://mongo:QybvGYWNoDKRrVbeZOxSSOfqgLrnkMDr@nozomi.proxy.rlwy.net:14169"
    }
    fmt.Println("Connecting to MongoDB at:", mongoUri)
    client, err := mongo.NewClient(options.Client().ApplyURI(mongoUri))
    if err != nil { log.Fatal(err) }
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    err = client.Connect(ctx)
    if err != nil { log.Fatal(err) }
    dbClient = client
    db = client.Database("teakspice")

    r := gin.Default()
    r.Use(cors.New(cors.Config{
    	AllowOrigins:     []string{"https://www.bestvragro.com"}, // use your real Vercel/custom domain!
    	AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    	AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    	AllowCredentials: true,
    }))


    // Auth
    r.POST("/api/register", register)
    r.POST("/api/login", login)

    // Products
    r.GET("/api/products", listProducts)

    // User
    auth := r.Group("/api", AuthMiddleware)
    {
        auth.GET("/user/profile", getProfile)
        auth.PUT("/user/profile", updateProfile)

        // Cart
        auth.GET("/cart", getCart)
        auth.POST("/cart", addToCart)
        auth.PUT("/cart/:productId", updateCart)
        auth.DELETE("/cart/:productId", removeCartItem)
        auth.POST("/cart/clear", clearCart)

        // Wishlist
        auth.GET("/wishlist", getWishlist)
        auth.POST("/wishlist", addToWishlist)
        auth.DELETE("/wishlist/:productId", removeFromWishlist)

        // Orders
        auth.GET("/orders", getOrders)
        auth.POST("/orders", placeOrder)
        auth.GET("/orders/:orderId", getOrder)
    }

    r.Run(":8080")
}

type JWTClaims struct {
    UserID string `json:"userId"`
    jwt.StandardClaims
}

func AuthMiddleware(c *gin.Context) {
    tokenStr := c.GetHeader("Authorization")
    if tokenStr == "" || len(tokenStr) < 8 {
        c.AbortWithStatusJSON(401, gin.H{"error": "missing token"})
        return
    }
    tokenStr = tokenStr[7:] // strip "Bearer "
    token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
        return jwtSecret, nil
    })
    if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
        c.Set("userId", claims.UserID)
        c.Next()
    } else {
        c.AbortWithStatusJSON(401, gin.H{"error": "invalid token", "detail": err.Error()})
    }
}

// ----- Auth -----

func register(c *gin.Context) {
    var req User
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": "invalid input"}); return }
    hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    req.Password = string(hashed)
    res, err := db.Collection("users").InsertOne(context.Background(), req)
    if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
    req.ID = res.InsertedID.(primitive.ObjectID)
    req.Password = "" // don't send back
    c.JSON(200, req)
}

func login(c *gin.Context) {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": "invalid input"}); return }
    var user User
    err := db.Collection("users").FindOne(context.Background(), bson.M{"email": req.Email}).Decode(&user)
    if err != nil { c.JSON(401, gin.H{"error": "user not found"}); return }
    if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
        c.JSON(401, gin.H{"error": "wrong password"}); return
    }
    // Generate JWT
    claims := JWTClaims{
        UserID: user.ID.Hex(),
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenStr, _ := token.SignedString(jwtSecret)
    user.Password = ""
    c.JSON(200, gin.H{"user": user, "token": tokenStr})
}

// ----- Products -----

func listProducts(c *gin.Context) {
    cur, err := db.Collection("products").Find(context.Background(), bson.M{})
    if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
    var products []Product
    if err := cur.All(context.Background(), &products); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, products)
}

// ----- Profile -----

func getProfile(c *gin.Context) {
    userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
    var user User
    err := db.Collection("users").FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
    user.Password = ""
    if err != nil { c.JSON(404, gin.H{"error": "not found"}); return }
    c.JSON(200, user)
}

func updateProfile(c *gin.Context) {
    userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
    var req User
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": "invalid input"}); return }
    update := bson.M{}
    if req.Name != "" { update["name"] = req.Name }
    if req.Email != "" { update["email"] = req.Email }
    if req.Phone != "" { update["phone"] = req.Phone }
    db.Collection("users").UpdateOne(context.Background(), bson.M{"_id": userID}, bson.M{"$set": update})
    getProfile(c)
}

// ----- Cart -----

func getCart(c *gin.Context) {
    userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
    var cart Cart
    db.Collection("carts").FindOne(context.Background(), bson.M{"userId": userID}).Decode(&cart)
    c.JSON(200, cart)
}

func addToCart(c *gin.Context) {
    userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
    var req CartItem
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": "invalid input"}); return }
    var cart Cart
    db.Collection("carts").FindOne(context.Background(), bson.M{"userId": userID}).Decode(&cart)
    found := false
    for i, item := range cart.Items {
        if item.ProductID == req.ProductID {
            cart.Items[i].Quantity += req.Quantity
            found = true
            break
        }
    }
    if !found {
        cart.Items = append(cart.Items, req)
    }
    db.Collection("carts").UpdateOne(context.Background(), bson.M{"userId": userID}, bson.M{"$set": bson.M{"items": cart.Items}}, options.Update().SetUpsert(true))
    c.JSON(200, cart)
}

func updateCart(c *gin.Context) {
    userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
    prodID, _ := primitive.ObjectIDFromHex(c.Param("productId"))
    var req struct{ Quantity int }
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": "invalid input"}); return }
    var cart Cart
    db.Collection("carts").FindOne(context.Background(), bson.M{"userId": userID}).Decode(&cart)
    for i, item := range cart.Items {
        if item.ProductID == prodID {
            if req.Quantity > 0 {
                cart.Items[i].Quantity = req.Quantity
            } else {
                // Remove
                cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
            }
            break
        }
    }
    db.Collection("carts").UpdateOne(context.Background(), bson.M{"userId": userID}, bson.M{"$set": bson.M{"items": cart.Items}})
    c.JSON(200, cart)
}

func removeCartItem(c *gin.Context) {
    userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
    prodID, _ := primitive.ObjectIDFromHex(c.Param("productId"))
    var cart Cart
    db.Collection("carts").FindOne(context.Background(), bson.M{"userId": userID}).Decode(&cart)
    for i, item := range cart.Items {
        if item.ProductID == prodID {
            cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
            break
        }
    }
    db.Collection("carts").UpdateOne(context.Background(), bson.M{"userId": userID}, bson.M{"$set": bson.M{"items": cart.Items}})
    c.JSON(200, cart)
}

func clearCart(c *gin.Context) {
    userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
    db.Collection("carts").UpdateOne(context.Background(), bson.M{"userId": userID}, bson.M{"$set": bson.M{"items": []CartItem{}}})
    c.JSON(200, gin.H{"status": "cleared"})
}

// ----- Wishlist -----

func getWishlist(c *gin.Context) {
    userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
    var w Wishlist
    db.Collection("wishlists").FindOne(context.Background(), bson.M{"userId": userID}).Decode(&w)
    c.JSON(200, w)
}

func addToWishlist(c *gin.Context) {
    userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
    var req struct{ ProductID primitive.ObjectID }
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": "invalid input"}); return }
    var w Wishlist
    db.Collection("wishlists").FindOne(context.Background(), bson.M{"userId": userID}).Decode(&w)
    for _, pid := range w.ProductIDs {
        if pid == req.ProductID {
            c.JSON(200, w)
            return
        }
    }
    w.ProductIDs = append(w.ProductIDs, req.ProductID)
    db.Collection("wishlists").UpdateOne(context.Background(), bson.M{"userId": userID}, bson.M{"$set": bson.M{"productIds": w.ProductIDs}}, options.Update().SetUpsert(true))
    c.JSON(200, w)
}

func removeFromWishlist(c *gin.Context) {
    userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
    prodID, _ := primitive.ObjectIDFromHex(c.Param("productId"))
    var w Wishlist
    db.Collection("wishlists").FindOne(context.Background(), bson.M{"userId": userID}).Decode(&w)
    for i, pid := range w.ProductIDs {
        if pid == prodID {
            w.ProductIDs = append(w.ProductIDs[:i], w.ProductIDs[i+1:]...)
            break
        }
    }
    db.Collection("wishlists").UpdateOne(context.Background(), bson.M{"userId": userID}, bson.M{"$set": bson.M{"productIds": w.ProductIDs}})
    c.JSON(200, w)
}

// ----- Orders -----

func getOrders(c *gin.Context) {
    userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
    cur, err := db.Collection("orders").Find(context.Background(), bson.M{"userId": userID})
    if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
    var orders []Order
    if err := cur.All(context.Background(), &orders); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, orders)
}

func getOrder(c *gin.Context) {
    userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
    orderID, _ := primitive.ObjectIDFromHex(c.Param("orderId"))
    var order Order
    err := db.Collection("orders").FindOne(context.Background(), bson.M{"userId": userID, "_id": orderID}).Decode(&order)
    if err != nil { c.JSON(404, gin.H{"error": "not found"}); return }
    c.JSON(200, order)
}

func placeOrder(c *gin.Context) {
    userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
    var req struct {
        Address string     `json:"address"`
        Items   []CartItem `json:"items"`
    }
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": "invalid input"}); return }
    
    // Step 1: Validate stock and calculate total
    total := 0
    var stockErrors []string
    
    for _, item := range req.Items {
        var product Product
        err := db.Collection("products").FindOne(context.Background(), bson.M{"_id": item.ProductID}).Decode(&product)
        if err != nil {
            c.JSON(400, gin.H{"error": "Product not found: " + item.ProductID.Hex()})
            return
        }
        
        // Check stock availability
        if product.Stock < item.Quantity {
            stockErrors = append(stockErrors, fmt.Sprintf("%s: requested %d, only %d available", product.Name, item.Quantity, product.Stock))
            continue
        }
        
        total += product.Price * item.Quantity
    }
    
    // If any stock validation errors, return them
    if len(stockErrors) > 0 {
        c.JSON(409, gin.H{"error": "Insufficient stock", "details": stockErrors})
        return
    }
    
    // Step 2: Create order first
    order := Order{
        ID:     primitive.NewObjectID(),
        UserID: userID,
        Address: req.Address,
        Items: req.Items,
        Total: total,
        Date: time.Now().Format(time.RFC3339),
        Status: "Processing",
    }
    
    _, err := db.Collection("orders").InsertOne(context.Background(), order)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to create order"})
        return
    }
    
    // Step 3: Update stock for each product (simple approach)
    for _, item := range req.Items {
        db.Collection("products").UpdateOne(
            context.Background(),
            bson.M{"_id": item.ProductID},
            bson.M{"$inc": bson.M{"stock": -item.Quantity}},
        )
    }
    
    // Step 4: Clear cart
    db.Collection("carts").UpdateOne(context.Background(), bson.M{"userId": userID}, bson.M{"$set": bson.M{"items": []CartItem{}}})
    
    c.JSON(200, gin.H{
        "success": true,
        "message": "Order placed successfully",
        "order": order,
        "stockUpdated": true,
    })
}
