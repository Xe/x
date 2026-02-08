package python_test

import (
	"context"
	"fmt"
	"log"

	"within.website/x/cmd/markdownlang/internal/python"
)

// Example_basic demonstrates basic Python code execution.
func Example_basic() {
	ctx := context.Background()
	result, err := python.Run(ctx, `print("Hello from Python!")`)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Stdout)
	// Output: Hello from Python!
}

// Example_math demonstrates mathematical calculations.
func Example_math() {
	ctx := context.Background()
	code := `
import math
# Calculate the area of a circle with radius 5
radius = 5
area = math.pi * radius ** 2
print(f"Area: {area:.2f}")
`
	result, err := python.Run(ctx, code)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(result.Stdout)
	// Output: Area: 78.54
}

// Example_json demonstrates JSON manipulation.
func Example_json() {
	ctx := context.Background()
	code := `
import json
data = {"name": "markdownlang", "version": "1.0"}
print(json.dumps(data))
`
	result, err := python.Run(ctx, code)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Stdout)
	// Output: {"name": "markdownlang", "version": "1.0"}
}

// Example_dataProcessing demonstrates data processing.
func Example_dataProcessing() {
	ctx := context.Background()
	code := `
# Process a list of numbers
numbers = [1, 2, 3, 4, 5]
squared = [x**2 for x in numbers]
sum_squared = sum(squared)
print(f"Numbers: {numbers}")
print(f"Squared: {squared}")
print(f"Sum of squares: {sum_squared}")
`
	result, err := python.Run(ctx, code)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(result.Stdout)
	// Output:
	// Numbers: [1, 2, 3, 4, 5]
	// Squared: [1, 4, 9, 16, 25]
	// Sum of squares: 55
}

// Example_errorHandling demonstrates error handling in Python code.
func Example_errorHandling() {
	ctx := context.Background()
	code := `
try:
    result = 10 / 0
except ZeroDivisionError as e:
    print(f"Caught error: {e}")
`
	result, err := python.Run(ctx, code)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Stdout)
	// Output: Caught error: division by zero
}
