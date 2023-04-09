package db

type Notification struct {
	Driver    string                 `json:"driver,omitempty"` // postgres
	TableName string                 `json:"table_name"`
	Operation string                 `json:"operation"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt string                 `json:"created_at"`
}

type Row struct {
	Table string
	Field string
	Type  interface{}
}

type Table map[string]Row

/*
	{
  "_id": "wBnqF2saqI",
  "locations": [
    {
      "_id": "r66tE5AWzm",
      "added": {
        "__type": "Date",
        "iso": "2023-04-09T09:35:23.544Z"
      },
      "label": "Al-Sheikh Tailor's, House 37, Road 4, Block E, Section 12",
      "latitude": "23.82351479999998",
      "longitude": "90.37590840000001",
      "floor": "-",
      "apartment": "-",
      "flat": "-",
      "name": "mohib",
      "phone": "01849239035",
      "city": "Dhaka",
      "postCode": "1215",
      "address": "Al-Sheikh Tailor's, House 37, Road 4, Block E, Section 12",
      "area": "Mirpur, Section 12"
    },
    {
      "_id": "YKUvGNF7Kl",
      "added": {
        "__type": "Date",
        "iso": "2023-04-09T09:36:46.690Z"
      },
      "label": "Farid Medicine Corner, House 73, Road 3, Block E, Section 12",
      "latitude": "23.823367007145293",
      "longitude": "90.37591144939904",
      "floor": "-",
      "apartment": "-",
      "flat": "-",
      "name": "mohib",
      "phone": "01849239035",
      "city": "Dhaka",
      "postCode": "1215",
      "address": "Farid Medicine Corner, House 73, Road 3, Block E, Section 12",
      "area": "Mirpur, Section 12"
    }
  ],
  "username": "01849239035",
  "type": "customer",
  "active": true,
  "promo": [],
  "nid_verified": false,
  "nid": [],
  "deactive_permission": true,
  "collection": 0,
  "rider_availability": false,
  "verified": false,
  "favorite_brands": [],
  "priorityOrder": 1,
  "_wperm": [
    "wBnqF2saqI"
  ],
  "_rperm": [
    "*",
    "wBnqF2saqI"
  ],
  "_auth_data_otp": {
    "id": "01849239035",
    "otp": "781810"
  },
  "_acl": {
    "wBnqF2saqI": {
      "w": true,
      "r": true
    },
    "*": {
      "r": true
    }
  },
  "_created_at": {
    "$date": {
      "$numberLong": "1678734526722"
    }
  },
  "_updated_at": {
    "$date": {
      "$numberLong": "1681033006714"
    }
  },
  "device_info": {
    "type": "Android"
  },
  "lastLogin": {
    "$date": {
      "$numberLong": "1681032170601"
    }
  },
  "push_token": "dd648GPQS9yGBGmhqsoFw3:APA91bGtMUdMPNibTziN6guSZ_ytBWr21f_FoXJ2eI-nVIrdbeusHjR2QNyBPGZLHo00zu9WVKqwDw27zFmfYpSoCOQXQU8_8cH0VA0im2pBPshU8l0jkf_eA_D7U7HjQbcsSLKPc9Ed",
  "name": "mohib",
  "note": "Name: Hamid Date :  29/03/23 Issue Details: Retarget // Calling time:   29/03/23, 3:58 p.m.  Call Summary:  Called customer 1x, Informed about ramadan deals  Resolution: Informed and provided voucher wc50 "
}
*/

func MongoSchema() {
	var tables = make(map[string]Table)

	tables["users"] = Table{
		"id": Row{
			Table: "users",
			Field: "id",
			Type:  "int",
		},
		"locations._id": Row{
			Table: "user_location",
			Field: "id",
			Type:  "int",
		},
		"locations.name": Row{},
	}
}
