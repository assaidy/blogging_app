# Blogging API

This project is a RESTful API for a Blogging platform.
It provides features for user authentication, user management,
post creation, commenting, reactions, bookmarks, notifications, and more.

## Features

### Authentication
- **User Registration**: Register a new user account.
- **User Login**: Authenticate and log in a user.
- **Access Tokens**: Retrieve access tokens for authenticated sessions.

### User Management
- **Get User by ID**: Fetch user details by their unique ID.
- **Get User by Username**: Fetch user details by their username.
- **Update User**: Update user profile information.
- **Delete User**: Delete a user account.
- **Get All Users**: Retrieve a list of all users with optional filtering for searching.

### Follow System
- **Follow User**: Follow another user.
- **Unfollow User**: Unfollow a user.
- **Get Followers**: Retrieve a list of followers for a specific user.

### Posts
- **Create Post**: Create a new post.
- **Get Post**: Fetch details of a specific post.
- **Update Post**: Update an existing post.
- **Delete Post**: Delete a post.
- **Get User Posts**: Retrieve all posts by a specific user.
- **Get All Posts**: Fetch all posts with optional filtering for searching.
- **View Post**: Record a view for a specific post.

### Comments
- **Create Comment**: Add a comment to a post.
- **Update Comment**: Edit an existing comment.
- **Delete Comment**: Remove a comment.
- **Get Post Comments**: Retrieve all comments for a specific post.

### Reactions
- **React to Post**: Add a reaction (like, dislike, etc.) to a post.
- **Delete Reaction**: Remove a reaction from a post.

### Bookmarks
- **Add to Bookmarks**: Bookmark a post for later viewing.
- **Remove from Bookmarks**: Remove a post from bookmarks.
- **Get All Bookmarks**: Retrieve all bookmarked posts for the authenticated user.

### Notifications
- **Get All Notifications**: Fetch all notifications for the authenticated user.
- **Get Unread Notifications Count**: Retrieve the count of unread notifications.
- **Mark Notification as Read**: Mark a specific notification as read.

---

## Getting Started

To get started with this project, follow these steps:

1. **Clone the repository**:
   ```bash
   git clone https://github.com/assaidy/blogging_app
   cd blogging_app
   ```

3. **Set up environment variables**:
   Rename `.env_example` to `.env` file in the root directory and change the necessary environment variables.

4. **Run containers**:
   ```bash
   make compose-up
   ```

5. **Migrate**:
  ```bash
  make goose-up
  ```

6. **Run the app**:
  ```bash
  make run
  ```

---

## Technologies Used
- **Fiber**: Web framework for building the API.
- **JWT (JSON Web Tokens)**: For user authentication.
- **PostgreSQL**: Relational database for data storage.
- **Cursor Pagination**: Implemented cursor-based pagination to minimize bandwidth usage and reduce server load,
 ensuring efficient data retrieval for large datasets.
