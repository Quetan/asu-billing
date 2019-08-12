package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func initializeDB() *sql.DB {
	dsn := fmt.Sprintf("%v:%v@tcp(10.0.0.33)/billingdev?parseTime=true", os.Getenv("MYSQL_LOGIN"), os.Getenv("MYSQL_PASS"))
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

// Payment struct
type Payment struct {
	Sum  int       `json:"sum"`
	Date time.Time `json:"date"`
}

// Tariff struct
type Tariff struct {
	ID    int    `json:"id"`
	Price int    `json:"price"`
	Name  string `json:"name"`
}

// User struct
type User struct {
	ID        uint      `json:"id"`
	Balance   int       `json:"balance"`
	Activity  bool      `json:"activity"`
	Name      string    `json:"name"`
	Room      string    `json:"room"`
	Login     string    `json:"login"`
	Phone     string    `json:"phone"`
	ExtIP     string    `json:"ext_ip"`
	InnerIP   string    `json:"inner_ip"`
	Tariff    Tariff    `json:"tariff"`
	Payments  []Payment `json:"payments,omitempty"`
	Agreement string    `json:"agreement"`
	// separate for a more beautiful view
	ExpiredDate     time.Time `json:"expired_date"`
	ConnectionPlace string    `json:"connection_place"`
}

// GetAllUsers returns all users from db
func GetAllUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(`SELECT users.id, balance, users.name, login, agreement, expired_date,
			connection_place, activity, room, phone, tariffs.id AS tariff_id, tariffs.name, price,
			ips.id, ext_ip
	FROM (( users
			INNER JOIN ips ON users.ip_id = ips.id)
			INNER JOIN tariffs ON users.tariff = tariffs.id)`)
	if err != nil {
		return nil, fmt.Errorf("cannot get all users: %v", err)
	}
	defer rows.Close()

	var user User
	users := make([]User, 0)
	for rows.Next() {
		err := rows.Scan(&user.ID, &user.Balance, &user.Name, &user.Login, &user.Agreement,
			&user.ExpiredDate, &user.ConnectionPlace, &user.Activity, &user.Room, &user.Phone,
			&user.Tariff.ID, &user.Tariff.Name, &user.Tariff.Price, &user.InnerIP, &user.ExtIP)
		if err != nil {
			return nil, fmt.Errorf("cannot get one row: %v", err)
		}
		users = append(users, user)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("something happened with rows: %v", err)
	}

	return users, nil
}

// GetUserByID returns user from db
func GetUserByID(db *sql.DB, id int) (User, error) {
	var user User
	err := db.QueryRow(`SELECT users.id, balance, users.name, login, agreement,
			expired_date, connection_place, activity, room, phone, tariffs.id AS tariff_id,
			tariffs.name AS tariff_name, price, ips.ip, ext_ip  
		FROM (( users
			INNER JOIN ips ON users.ip_id = ips.id)
			INNER JOIN tariffs ON users.tariff = tariffs.id)
		WHERE users.id = ?`, id).Scan(&user.ID, &user.Balance, &user.Name, &user.Login,
		&user.Agreement, &user.ExpiredDate, &user.ConnectionPlace, &user.Activity, &user.Room,
		&user.Phone, &user.Tariff.ID, &user.Tariff.Name, &user.Tariff.Price, &user.InnerIP, &user.ExtIP)
	if err != nil {
		return user, fmt.Errorf("cannot get user with id=%v: %v", id, err)
	}

	return user, nil
}

// GetUserIDbyLogin returns user id from db
func GetUserIDbyLogin(db *sql.DB, login string) (uint, error) {
	var id uint
	err := db.QueryRow(`SELECT id FROM users WHERE login = ?`, login).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("cannot get user id by login: %v", err)
	}

	return id, nil
}

func AddUserToDB(db *sql.DB, user User) (int, error) {
	innerIPid, err := getUnusedInnerIPid(db)
	if err != nil {
		return 0, fmt.Errorf("cannot get unused id of inner ip: %v", err)
	}

	res, err := db.Exec(`INSERT INTO users (balance, activity, name, room, login, phone, ext_ip,
		ip_id, tariff, agreement, connection_place, expired_date) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		user.Balance, user.Activity, user.Name, user.Room, user.Login, user.Phone, "82.200.46.10",
		innerIPid, user.Tariff.ID, user.Agreement, user.ConnectionPlace, user.ExpiredDate)
	if err != nil {
		return 0, fmt.Errorf("cannot insert values in db: %v", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("cannot get last insert id: %v", err)
	}

	return int(id), nil
}

func getUnusedInnerIPid(db *sql.DB) (int, error) {
	var innerIPid int
	err := db.QueryRow(`SELECT id FROM ips WHERE used = 0`).Scan(&innerIPid)
	if err != nil {
		return 0, fmt.Errorf("cannot scan id from db to variable: %v", err)
	}

	_, err = db.Exec(`UPDATE ips SET used=1 WHERE id = ?`, innerIPid)
	if err != nil {
		return 0, fmt.Errorf("cannot set inner ip as used: %v", err)
	}

	return innerIPid, nil
}

// func DeleteUserByID(db *sql.DB, id int) error {
// 	var innerIPid int
// 	err := db.QueryRow(`SELECT ip_id FROM users WHERE id = ?`, id).Scan(&innerIPid)
// 	if err != nil {
// 		return fmt.Errorf("could not scan ip_id from users table to variable: %v", err)
// 	}

// 	_, err = db.Exec(`UPDATE ips SET used = 0 WHERE id=?`, innerIPid)
// 	if err != nil {
// 		return fmt.Errorf("could not set false used state to ip_id: %v", err)
// 	}

// 	_, err = db.Exec(`DELETE FROM users WHERE id = ?`, id)
// 	if err != nil {
// 		return fmt.Errorf("could not delete user with id=%v: %v", id, err)
// 	}

// 	_, err = db.Exec(`DELETE FROM payments WHERE user_id = ?`, id)
// 	if err != nil {
// 		return fmt.Errorf("could not delete payments info about user with id=%v: %v", id, err)
// 	}

// 	return nil
// }
