// User object returned by the API
export interface User {
  Username: string;
  DisplayName?: string;
}

// User object sent to the API when registering/editing a user
export interface UserWithPassword {
  Username: string;
  Password: string;
  DisplayName?: string;
}

export interface Blog {
  Slug: string;
  DisplayName?: string;
}
