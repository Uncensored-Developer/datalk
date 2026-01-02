package common

func BoolFromDB(dbBool int64) bool {
	return dbBool == 1
}

func BoolToDB(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
