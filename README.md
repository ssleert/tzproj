# tzproj

## `/put` endpoint
input:
```json
{
  "name": "Alex",
  "surname": "Ryabinkov",
  "patronymic": "Pavlovich" // optional
}
```
output:
```json
{
  "writed_id":1,
  "error":"null"
}
```

## `/del` endpoint
input:
```json
{
  "delete_id": 1
}
```
output:
```json
{
  "error":"null"
}
```

## `/update` endpoint
input:
```json
{
  "people": {
    "id": 1,
    "age": 136,
    "name": "Simon",
    "surname": "Pavlovich",
    "patronymic": "Alexandrovich" // optional
  },
  "gender": {
    "gender": "male",
    "probability": 0.123123
  },
  "nationalizations": [
    {
      "country_code": "RU",
      "probability": 0.92
    }
  ]
}
```
output:
```json
{
  "error":"null"
}
```

## `/get` endpoint
input:
```json
{
  "offset": 0,
  "limit": 10,
  "filter_by": [
    {
      "key": "name",
      "op": "=",
      "value": "Simon"
    }
  ]
}
```
output:
```json
{
  "data": [
    {
      "people": {
        "id": 2,
        "name": "Simon",
        "surname": "Pavlovich",
        "patronymic": "Alexandrovich",
        "age": 136
      },
      "gender": {
        "gender": "male",
        "probability": 0.123123
      },
      "nationalizations": [
        {
          "country_code": "RU",
          "probability": 0.92
        }
      ]
    }
  ],
  "error": "null"
}
```
