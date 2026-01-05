ef divide(a, b):
    try:
        result = a / b
        return result
    except ZeroDivisionError:
        print("Error: Division by zero!")
        return None
    except TypeError:
        print("Error: Invalid types!")
        return None
    finally:
        print("Division operation completed")

# File operations
def main():
    # Write to file
    with open("example.txt", "w") as f:
        f.write("Hello, World!\n")
        f.write("This is a test file.\n")

    # Read from file
    try:
        with open("example.txt", "r") as f:
            content = f.read()
            print("File content:")
            print(content)
    except FileNotFoundError:
        print("File not found!")

# Dictionary operations
def main():
    # Create dictionary
    person = {
        "name": "Alice",
        "age": 25,
        "city": "New York"
    }

    # Access values
    print(f"Name: {person['name']}")
    print(f"Age: {person.get('age')}")

    # Add/update values
    person["email"] = "alice@example.com"
    person["age"] = 26

    # Iterate over dictionary
    for key, value in person.items():
        print(f"{key}: {value}")

# JSON handling
import json

def main():
    # Create data
    data = {
        "name": "Bob",
        "age": 30,
        "hobbies": ["reading", "coding", "gaming"]
    }

    # Write JSON to file
    with open("data.json", "w") as f:
        json.dump(data, f, indent=4)

    # Read JSON from file
    with open("data.json", "r") as f:
        loaded_data = json.load(f)
        print("Loaded data:", loaded_data)

# Lambda functions
def main():
    # Simple lambda
    square = lambda x: x**2
    print("Square of 5:", square(5))

    # Lambda with multiple arguments
    add = lambda x, y: x + y
    print("5 + 3 =", add(5, 3))

    # Lambda in sorting
    points = [(1, 2), (3, 1), (5, 0), (0, 4)]
    points.sort(key=lambda p: p[1])  # Sort by y-coordinate
    print("Sorted by y:", points)

# Decorators
def timer_decorator(func):
    def wrapper(*args, **kwargs):
        import time
        start_time = time.time()
        result = func(*args, **kwargs)
        end_time = time.time()
        print(f"{func.__name__} took {end_time - start_time:.2f} seconds")
        return result
    return wrapper

@timer_decorator
def slow_function():
    import time
    time.sleep(1)
    return "Done!"

# Context managers
class FileManager:
    def __init__(self, filename, mode):
        self.filename = filename
        self.mode = mode
        self.file = None

    def __enter__(self):
        self.file = open(self.filename, self.mode)
        return self.file

    def __exit__(self, exc_type, exc_val, exc_tb):
        if self.file:
            self.file.close()
