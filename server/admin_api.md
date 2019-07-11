# Admin API

The Admin API uses JSON-RPC. Each call should be of the form

```
{
  "id": "0001",
  "method": "Service.Method",
  "params": [{"param1": "value1", "param2": "value2"}]
}
```

`id` is a unique unsigned integer used to match responses with requests. The
parameters object is always wrapped in a 1-sized array.

## User management

### Users.Create

Authentication required: none

Parameters:

```
{
  "username": string, // required, [a-zA-Z][a-zA-Z.-_]+ , unique
  "password": string, // required, min 6 characters
  "displayName": string
}
```

Response:

```
{}
```

### Users.Update

Authentication required: valid-user

Parameters:

```
{
  "password": string,
  "displayName": string
}
```

### Users.Get

Authentication required: none

Parameters:

```
{
  "username": string // required
}
```

Response:

```
{
  "username": string,
  "displayName": string,
  "blogs": ["blog1", "blog2"]
}
```

### Users.Delete

Authentication required: valid-user

Parameters:

```
{
  "username": string // required
}
```

Response:

```
{}
```

## Blog management

### Blogs.Create

Authentication required: valid-user

Parameters:

```
{
  "slug": string, // required, unique, cannot be changed
  "title": string // required
}
```

Response:

```
{}
```

### Blogs.GetForUser

Authentication required: none

Parameters:

```
{
  "username": string, // required
}
```

Response:

```
{
  "blogs": [
    {
	  "slug": string,
	  "title": string
	},
	...
  ]
}
```

### Blogs.Update

Authentication required: valid-user

Parameters:

```
{
  "slug": string, // required, used to identify the blog
  "title": string // required
}
```

Response:

```
{}
```

### Blogs.Delete

Authentication required: valid-user

Parameters:

```
{
  "slug": string, // required, used to identify the blog
}
```
