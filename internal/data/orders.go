package data

import (
	"context"
	"database/sql"
	"errors"
	"finalproject/internal/validator"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"time"
)

type OrderItem struct {
	ProductId int64 `json:"productId"`
	Quantity  int   `json:"quantity"`
}

type Order struct {
	ID         int64       `json:"id"`
	OrderedAt  time.Time   `json:"-"`
	Status     int         `json:"status"`
	Address    string      `json:"address"`
	UserId     int64       `json:"userId"`
	TotalPrice float32     `json:"totalPrice"`
	OrderItems []OrderItem `json:"orderItems"`
	Version    int         `json:"version"`
}

func ValidateOrder(v *validator.Validator, order *Order) {
	v.Check(order.Address != "", "address", "must be provided")
	v.Check(order.UserId >= 0, "user", "must be provided")
	//v.Check(order.TotalPrice != 0, "total_price", "must be provided")
	//v.Check(order.TotalPrice > 0, "total_price", "must be a positive value")
	v.Check(order.OrderItems != nil, "order_items", "must be provided")
	v.Check(len(order.OrderItems) >= 1, "order_items", "must contain at least 1 order item")

}
func ValidateUpdatedOrder(v *validator.Validator, order *Order) {
	v.Check(order.Address != "", "address", "must be provided")
	v.Check(order.UserId >= 0, "user", "must be provided")
	v.Check(order.Status >= 0, "status", "must be greater or equal to zero")
	v.Check(order.TotalPrice != 0, "total_price", "must be provided")
	v.Check(order.TotalPrice > 0, "total_price", "must be a positive value")
}

type OrderModel struct {
	DB *pgxpool.Pool
}

func (m OrderModel) Insert(userId int64, order *Order, r *http.Request) error {
	totalPrice := 0.00
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	for _, item := range order.OrderItems {
		if item.ProductId < 1 {
			return ErrRecordNotFound
		}
		// Define the SQL query for retrieving the product data.
		query := `SELECT product_id, created_at, title, owner, description, images, colors, quantity, price, version
				FROM products
					WHERE product_id = $1`
		var product Product

		err := m.DB.QueryRow(ctx, query, item.ProductId).Scan(
			&product.ID,
			&product.CreatedAt,
			&product.Title,
			&product.Owner,
			&product.Description,
			&product.Images,
			&product.Colors,
			&product.Quantity,
			&product.Price,
			&product.Version,
		)

		if err != nil {
			switch {
			case errors.Is(err, sql.ErrNoRows):
				return ErrRecordNotFound
			case errors.Is(err, pgx.ErrNoRows):
				return ErrRecordNotFound
			default:
				return err
			}
		}
		query = `
		UPDATE products
		SET quantity = quantity - $1 , version = version + 1
		WHERE product_id = $2 AND version = $3
		RETURNING version`
		err = m.DB.QueryRow(ctx, query, item.Quantity, product.ID, product.Version).Scan(&product.Version)
		if err != nil {
			switch {
			case err.Error() == `ERROR: new row for relation "products" violates check constraint "quantity_check" (SQLSTATE 23514)`:
				return ErrOutOfStock
			case errors.Is(err, pgx.ErrNoRows):
				return ErrRecordNotFound
			default:
				return err
			}
		}
		totalPrice += product.Price + float64(item.Quantity)
	}
	query := `INSERT INTO orders (user_id, total_price, address) VALUES ($1, $2, $3) 
                                                  RETURNING id, ordered_at, user_id`
	err := m.DB.QueryRow(ctx, query, userId, totalPrice, order.Address).Scan(&order.ID, &order.OrderedAt, &order.UserId)
	if err != nil {
		return err
	}
	order.TotalPrice = float32(totalPrice)
	var tempId int
	query = `INSERT INTO order_items (order_id, product_id, quantity) VALUES ($1, $2, $3) returning id`
	for i := range order.OrderItems {
		err = m.DB.QueryRow(ctx, query, order.ID, order.OrderItems[i].ProductId, order.OrderItems[i].Quantity).Scan(&tempId)
		if err != nil {
			return err
		}
	}
	return nil
}
func (m OrderModel) Get(id int64, r *http.Request) (*Order, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `SELECT id,user_id, ordered_at, status, total_price, address, version
				FROM orders
					WHERE id = $1`
	var order Order
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	// Importantly, use defer to make sure that we cancel the context before the Get()
	// method returns.
	defer cancel()

	err := m.DB.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.UserId,
		&order.OrderedAt,
		&order.Status,
		&order.TotalPrice,
		&order.Address,
		&order.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	// Otherwise, return a pointer to the Movie struct.
	return &order, nil
}
func (m OrderModel) GetAllOrdersForUser(userId int64, filters Filters, r *http.Request) ([]*Order, Metadata, error) {
	if userId < 1 {
		return nil, Metadata{}, ErrRecordNotFound
	}
	// Define the SQL query for retrieving the order data.
	query := fmt.Sprintf(`SELECT count(*) over (),id, ordered_at, status, total_price, address
				FROM orders
					WHERE user_id = $1 ORDER BY %s %s LIMIT $2 OFFSET $3`, filters.sortColumn(), filters.sortDirection())
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	// Importantly, use defer to make sure that we cancel the context before the Get()
	// method returns.
	defer cancel()

	rows, err := m.DB.Query(ctx, query, userId, filters.limit(), filters.offset())
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()
	totalRecords := 0
	orders := []*Order{}
	for rows.Next() {
		var order Order
		err := rows.Scan(
			&totalRecords,
			&order.ID,
			&order.OrderedAt,
			&order.Status,
			&order.TotalPrice,
			&order.Address,
		)

		if err != nil {
			return nil, Metadata{}, err
		}
		orders = append(orders, &order)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	query = `SELECT product_id, quantity
				FROM order_items
					WHERE order_id = $1`

	for i, loopingOrder := range orders {
		otherRows, err := m.DB.Query(ctx, query, loopingOrder.ID)
		if err != nil {
			return nil, Metadata{}, err
		}
		for otherRows.Next() {
			var orderItem OrderItem
			err := otherRows.Scan(
				&orderItem.ProductId,
				&orderItem.Quantity,
			)

			if err != nil {
				return nil, Metadata{}, err
			}
			orders[i].OrderItems = append(orders[i].OrderItems, orderItem)
		}
		if err = otherRows.Err(); err != nil {
			return nil, Metadata{}, err
		}
	}
	return orders, metadata, nil
}

// Update status of an order
func (m OrderModel) Update(order *Order, r *http.Request) error {
	query := `
		UPDATE orders
			SET status = $1 WHERE id = $2 AND version = $3
		RETURNING version`
	// Create an args slice containing the values for the placeholder parameters.
	args := []any{
		&order.Status,
		&order.ID,
		&order.Version,
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRow(ctx, query, args...).Scan(&order.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		case errors.Is(err, pgx.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

func (m OrderModel) Delete(id int64, r *http.Request) error {
	// Return an ErrRecordNotFound error if the product ID is less than 1.
	if id < 1 {
		return pgx.ErrNoRows
	}
	// Construct the SQL query to delete the record.
	query := `
		DELETE FROM orders
			WHERE id = $1 returning user_id`
	// Execute the SQL query using the Exec() method, passing in the id variable as
	// the value for the placeholder parameter. The Exec() method returns a sql.Result
	// object.
	var order Order
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	rows := m.DB.QueryRow(ctx, query, id)
	err := rows.Scan(
		&order.UserId,
	)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrRecordNotFound
		case errors.Is(err, sql.ErrNoRows):
			return ErrRecordNotFound
		default:
			return err
		}
	}

	return nil
}

func (m OrderModel) IsUserOrderedProduct(userId int64, productId int64, r *http.Request) (bool, error) {
	if productId < 1 {
		return false, ErrRecordNotFound
	}
	query := `Select exists(SELECT orders.id, orders.ordered_at, orders.status, orders.total_price, orders.address
              FROM orders
                       JOIN order_items ON orders.id = order_items.order_id
              WHERE orders.user_id = 3
                AND order_items.product_id = 9);`
	var isOrdered bool
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	// Importantly, use defer to make sure that we cancel the context before the Get()
	// method returns.
	defer cancel()

	err := m.DB.QueryRow(ctx, query, userId, productId).Scan(
		&isOrdered,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return false, ErrRecordNotFound
		case errors.Is(err, pgx.ErrNoRows):
			return false, ErrRecordNotFound
		default:
			return false, err
		}
	}
	// Otherwise, return a pointer to the Movie struct.
	return isOrdered, nil
}
