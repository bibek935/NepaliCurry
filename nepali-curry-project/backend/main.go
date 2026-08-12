package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type MenuItem struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Price       int     `json:"price"`
	Description string  `json:"description"`
	ImageURL    string  `json:"image_url"`
}

type OrderItemReq struct {
	MenuItemID int `json:"menu_item_id"`
	Quantity   int `json:"quantity"`
}

type OrderReq struct {
	TableNo int            `json:"table_no"`
	Items   []OrderItemReq `json:"items"`
}

type OrderItemDetail struct {
	ID         int    `json:"id"`
	MenuItemID int    `json:"menu_item_id"`
	Name       string `json:"name"`
	Price      int    `json:"price"`
	Quantity   int    `json:"quantity"`
}

type Order struct {
	ID        int               `json:"id"`
	TableNo   int               `json:"table_no"`
	Status    string            `json:"status"` // "ordered", "paid"
	Total     int               `json:"total"`
	CreatedAt time.Time         `json:"created_at"`
	Items     []OrderItemDetail `json:"items"`
}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./nepali_curry.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	initDB()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/menu", handleMenu)
	mux.HandleFunc("/api/orders", handleOrders)
	mux.HandleFunc("/api/orders/pay", handlePay)

	corsMux := corsMiddleware(mux)

	log.Println("サーバーをポート8080で起動中...")
	if err := http.ListenAndServe(":8080", corsMux); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func initDB() {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS menu_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			category TEXT,
			price INTEGER,
			description TEXT,
			image_url TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			table_no INTEGER,
			status TEXT,
			total INTEGER,
			created_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS order_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER,
			menu_item_id INTEGER,
			quantity INTEGER
		);`,
	}

	for _, stmt := range statements {
		_, err := db.Exec(stmt)
		if err != nil {
			log.Fatal(err)
		}
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM menu_items").Scan(&count)
	if count == 0 {
		db.Exec(`INSERT INTO menu_items (name, category, price, description, image_url) VALUES 
			('Pannirー', 'Curry', 900, 'スパイシーな伝統的ネパールチキンカレー', 'https://images.unsplash.com/photo-1588166524941-3bf61a9c41db?w=500'),
			('マトンカレー', 'Curry', 1050, 'じっくり煮込んだコクのある羊肉カレー', 'https://images.unsplash.com/photo-1545247181-516773cae754?w=500'),
			('leg piece', 'Nan/Rice', 300, '焼きたてもちもちのナン', 'https://images.unsplash.com/photo-1626074353765-517a681e40be?w=500'),
			('SAMOSA', 'Nan/Rice', 500, 'とろーりチーズがたっぷり詰まったナン', 'https://images.unsplash.com/photo-1601050690597-df0568f70950?w=500'),
			('モモ (6個)', 'Side', 600, 'ネパール風蒸し餃子特製ソース添え', 'https://images.unsplash.com/photo-1625220194771-7ebdea0b70b9?w=500'),
			('Mango Lassi', 'Drink', 350, '濃厚なマンゴー味のヨーグルトドリンク', 'https://images.unsplash.com/photo-1571006682858-a458b8d29f8f?w=500')`)
	}
}

func handleMenu(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query("SELECT id, name, category, price, description, image_url FROM menu_items")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []MenuItem
	for rows.Next() {
		var item MenuItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Price, &item.Description, &item.ImageURL); err == nil {
			items = append(items, item)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT id, table_no, status, total, created_at FROM orders ORDER BY created_at DESC")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var orders []Order
		for rows.Next() {
			var o Order
			if err := rows.Scan(&o.ID, &o.TableNo, &o.Status, &o.Total, &o.CreatedAt); err != nil {
				continue
			}

			itemRows, err := db.Query(`
				SELECT oi.id, oi.menu_item_id, m.name, m.price, oi.quantity 
				FROM order_items oi 
				JOIN menu_items m ON oi.menu_item_id = m.id 
				WHERE oi.order_id = ?`, o.ID)

			o.Items = []OrderItemDetail{}
			if err == nil {
				for itemRows.Next() {
					var item OrderItemDetail
					itemRows.Scan(&item.ID, &item.MenuItemID, &item.Name, &item.Price, &item.Quantity)
					o.Items = append(o.Items, item)
				}
				itemRows.Close()
			}
			orders = append(orders, o)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(orders)

	} else if r.Method == "POST" {
		var req OrderReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Items) == 0 {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		total := 0
		type calcItem struct {
			menuItemID int
			quantity   int
			price      int
		}
		var calcItems []calcItem

		for _, item := range req.Items {
			var price int
			err := db.QueryRow("SELECT price FROM menu_items WHERE id = ?", item.MenuItemID).Scan(&price)
			if err != nil {
				http.Error(w, "Item not found", http.StatusBadRequest)
				return
			}
			total += price * item.Quantity
			calcItems = append(calcItems, calcItem{menuItemID: item.MenuItemID, quantity: item.Quantity, price: price})
		}

		res, err := db.Exec("INSERT INTO orders (table_no, status, total, created_at) VALUES (?, 'ordered', ?, ?)",
			req.TableNo, total, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		orderID, _ := res.LastInsertId()
		for _, item := range calcItems {
			db.Exec("INSERT INTO order_items (order_id, menu_item_id, quantity) VALUES (?, ?, ?)",
				orderID, item.menuItemID, item.quantity)
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func handlePay(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID int `json:"order_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE orders SET status = 'paid' WHERE id = ?", req.OrderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}