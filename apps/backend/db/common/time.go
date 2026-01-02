package common

import "time"

func TimeFromDB(dbTime string) time.Time {
	if dbTime == "" {
		return time.Time{}
	}
	layout := "2006-01-02 15:04:05.000"

	t, err := time.ParseInLocation(layout, dbTime, time.UTC)
	if err != nil {
		panic(err)
	}
	return t
}

func TimeToDB(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}
