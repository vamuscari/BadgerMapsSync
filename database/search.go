package database

import (
	"badgermaps/api"
	"fmt"
	"strings"
)

type CheckinRow struct {
	CheckinId   int
	AccountId   int
	AccountName string
	LogDatetime string
}

func SearchAccounts(db DB, q string) ([]api.Account, error) {
	sqlText := db.GetSQL("SearchAccounts")
	if sqlText == "" {
		return nil, fmt.Errorf("unknown or unavailable SQL command: SearchAccounts")
	}
	like := "%" + q + "%"
	prefix := q + "%"
	word := "% " + q + "%"
	rows, err := db.GetDB().Query(sqlText,
		like,                    // WHERE name
		like,                    // WHERE id cast
		q,                       // rank exact id
		q,                       // rank exact name
		strings.ToLower(prefix), // rank prefix
		strings.ToLower(word),   // rank word
		strings.ToLower(like),   // rank substring
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []api.Account
	for rows.Next() {
		var a api.Account
		if err := rows.Scan(&a.AccountId, &a.FullName); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

func SearchRoutes(db DB, q string) ([]api.Route, error) {
	sqlText := db.GetSQL("SearchRoutes")
	if sqlText == "" {
		return nil, fmt.Errorf("unknown or unavailable SQL command: SearchRoutes")
	}
	like := "%" + q + "%"
	prefix := q + "%"
	word := "% " + q + "%"
	rows, err := db.GetDB().Query(sqlText,
		like, like, q, q, strings.ToLower(prefix), strings.ToLower(word), strings.ToLower(like),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []api.Route
	for rows.Next() {
		var r api.Route
		if err := rows.Scan(&r.RouteId, &r.Name, &r.RouteDate); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

func SearchCheckins(db DB, q string) ([]CheckinRow, error) {
	sqlText := db.GetSQL("SearchCheckins")
	if sqlText == "" {
		return nil, fmt.Errorf("unknown or unavailable SQL command: SearchCheckins")
	}
	like := "%" + q + "%"
	prefix := q + "%"
	word := "% " + q + "%"
	rows, err := db.GetDB().Query(sqlText,
		like, like, q, q, strings.ToLower(prefix), strings.ToLower(word), strings.ToLower(like),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []CheckinRow
	for rows.Next() {
		var c CheckinRow
		if err := rows.Scan(&c.CheckinId, &c.AccountId, &c.AccountName, &c.LogDatetime); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}
