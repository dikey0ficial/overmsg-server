package main

// User is for users in
// db
type User struct {
	Token string `bson:"_id"`
	Name  string `bson:"name"`
}

// Answer is type for JSON answer
type Answer struct {
	Success bool   `json:"succes"`
	Error   string `json:"error"`
	Res     Result `json:"result"`
}

// Result needs to use
// in Answer struct
type Result interface {
	Result()
}

// ToJSON returns encoded answer
// as bytes
func (a Answer) ToJSON() []byte {
	res, err := json.Marshal(a)
	if err != nil {
		infl.Println("[ERROR] answer2json: ", err)
		return []byte(`{"succes":false,"error":"server error"}`)
	}
	return res
}

// AuthResult is result for Auth
type AuthResult struct {
	Token string `json:"token"`
}

// Result method for Result interface
func (AuthResult) Result() {}

// Message is message.
type Message struct {
	From    string `json:"from_name"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message"`
}

// ToJSON returns encoded message
// as bytes
func (m Message) ToJSON() []byte {
	res, err := json.Marshal(m)
	if err != nil {
		infl.Println("[ERROR] message2json: ", err)
		return []byte(`{"error":"Error of encoding"}`)
	}
	return res
}

// SendMessageRequest is for
// getting data from SendMessage
// request
// I added this because
// it can have a lot of values
type SendMessageRequest struct {
	PeerName string `json:"peer_name"`
	Message  string `json:"message"`
}
