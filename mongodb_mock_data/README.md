# MongoDB Mock Data

This folder contains JSON files with mock data for your MongoDB Atlas database.

## Files

- `products.json` - Contains 15 mock products for your e-commerce store

## How to Import to MongoDB Atlas

1. Log in to your MongoDB Atlas account
2. Navigate to your cluster and click "Browse Collections"
3. Select your database (`teakspice`)
4. Click "Add My Own Data" or use the "Import" button
5. Choose the collection name (`products`)
6. Upload the `products.json` file

## Product Data Structure

Each product contains:
- `name`: Product name
- `price`: Price in cents/paise (integer)
- `image`: Product image URL
- `description`: Product description
- `stock`: Available stock quantity

The data matches your Go Product model structure from `models.go`.