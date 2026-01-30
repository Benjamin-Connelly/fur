#!/usr/bin/env python3
"""
Test Python file with syntax highlighting
Demonstrates various Python features
"""

import os
import sys
from typing import List, Optional, Dict
from dataclasses import dataclass


@dataclass
class User:
    """Represents a user in the system"""
    id: int
    username: str
    email: str
    is_active: bool = True


class UserManager:
    """Manages user operations"""

    def __init__(self, db_connection: str):
        self.db_connection = db_connection
        self.users: Dict[int, User] = {}

    def create_user(self, username: str, email: str) -> User:
        """
        Create a new user

        Args:
            username: The username for the new user
            email: The email address for the new user

        Returns:
            The newly created User object
        """
        user_id = len(self.users) + 1
        user = User(id=user_id, username=username, email=email)
        self.users[user_id] = user
        return user

    def find_user(self, user_id: int) -> Optional[User]:
        """Find a user by ID"""
        return self.users.get(user_id)

    def get_active_users(self) -> List[User]:
        """Get all active users"""
        return [user for user in self.users.values() if user.is_active]


def fibonacci(n: int) -> int:
    """Calculate the nth Fibonacci number"""
    if n <= 1:
        return n
    return fibonacci(n - 1) + fibonacci(n - 2)


def main():
    """Main function"""
    manager = UserManager("postgresql://localhost/mydb")

    # Create some users
    user1 = manager.create_user("alice", "alice@example.com")
    user2 = manager.create_user("bob", "bob@example.com")

    print(f"Created users: {user1.username}, {user2.username}")

    # Calculate Fibonacci
    for i in range(10):
        print(f"F({i}) = {fibonacci(i)}")


if __name__ == "__main__":
    main()
