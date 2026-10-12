package mapper

import "time"

func databaseTime(value time.Time) time.Time {
	return value.UTC().Truncate(time.Microsecond)
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func StringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func OptionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	utc := databaseTime(value)
	return &utc
}

func timeValue(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}

	return value.UTC()
}
