package main

// User is for users in db
type User struct {
	Name  string `bson:"name"`
	Pass  string `bson:"pass"`
	Token string `bson:"_id"`
}

// Answer is type for JSON answer
type Answer struct {
	Success bool   `json:"succes"`
	Error   string `json:"error,omitempty"`
	Res     Result `json:"result,omitempty"`
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
		errl.Println("answer2json: ", err)
		return []byte(`{"succes":false,"error":"server error"}`)
	}
	return res
}

// TokenResult is result for reg and get_token
type TokenResult struct {
	Token string `json:"token"`
}

// Result method for Result interface
func (TokenResult) Result() {}

// IsOnlineResult is result for IsOnline
type IsOnlineResult struct {
	Is bool `json:"is"`
}

// Result method for Result interface
func (IsOnlineResult) Result() {}

// Message is message.
type Message struct {
	From    string `json:"from_name"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
	Type    string `json:"type"`
}

// ToJSON returns encoded message
// as bytes
func (m Message) ToJSON() []byte {
	m.Type = "message"
	res, err := json.Marshal(m)
	if err != nil {
		infl.Println("[ERROR] message2json: ", err)
		return []byte(`{"type":"message","error":"Error of encoding"}`)
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
