package data

import (
	"database/sql"
	"sync"
)

type MuxDb struct {
	db *sql.DB
	sync.RWMutex
}

func (m *MuxDb) Close() {
	m.RLock()
	defer m.RUnlock()
	m.db.Close()
}

func (m *MuxDb) Do(cb DoFn) error {
	m.RLock()
	defer m.RUnlock()

	// // list all tables within the database and print them
	// rows, err := m.Db.Query("SELECT name FROM sqlite_master WHERE type='table';")
	// if err != nil {
	// 	fmt.Println("failed", err)
	// 	return err
	// }
	// defer rows.Close()
	// count := 0
	// for rows.Next() {
	// 	var name string
	// 	if err := rows.Scan(&name); err != nil {
	// 		return err
	// 	}
	// 	count++
	// 	fmt.Println(name)
	// }
	// fmt.Println("count tables", count)

	return cb(m.db)
}

func NewMuxDb(db *sql.DB) *MuxDb {
	return &MuxDb{db: db}
}
