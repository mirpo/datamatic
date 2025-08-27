package retry

type testHTTPError struct {
	statusCode int
	message    string
}

func (e *testHTTPError) Error() string {
	return e.message
}

func (e *testHTTPError) HTTPStatusCode() int {
	return e.statusCode
}
