package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SampleProblem represents a problem to seed
type SampleProblem struct {
	Name             string
	Difficulty       string
	TimeLimit        float64
	MemoryLimit      int32
	Points           int32
	ProblemStatement string
	TestCases        []SampleTestCase
}

// SampleTestCase represents a test case
type SampleTestCase struct {
	Rank        int32
	IsSample    bool
	Input       string
	Output      string
	Description string
}

// SampleContest represents a contest to seed
type SampleContest struct {
	ExternalID   string
	Name         string
	ShortName    string
	StartTime    time.Time
	EndTime      time.Time
	ProblemNames []string
	ShortNames   []string
	Points       []int32
}

// Sample problems data
var sampleProblems = []SampleProblem{
	{
		Name:        "A + B",
		Difficulty:  "easy",
		TimeLimit:   1.0,
		MemoryLimit: 262144, // 256 MB
		Points:      100,
		ProblemStatement: `
<h1>A + B</h1>
<h2>Problem Description</h2>
<p>Given two integers A and B, your task is to calculate their sum.</p>
<h2>Input</h2>
<p>The input consists of a single line containing two integers A and B, separated by a space.</p>
<p>Constraints:</p>
<ul>
<li>-10<sup>9</sup> ≤ A, B ≤ 10<sup>9</sup></li>
</ul>
<h2>Output</h2>
<p>Output a single line containing the sum of A and B.</p>
<h2>Sample Input 1</h2>
<pre>1 2</pre>
<h2>Sample Output 1</h2>
<pre>3</pre>
<h2>Sample Input 2</h2>
<pre>-5 10</pre>
<h2>Sample Output 2</h2>
<pre>5</pre>
<h2>Hint</h2>
<p>This is the simplest problem. Just add the two numbers together.</p>
`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "1 2\n", Output: "3\n", Description: "Simple positive numbers"},
			{Rank: 2, IsSample: true, Input: "-5 10\n", Output: "5\n", Description: "Mixed positive and negative"},
			{Rank: 3, IsSample: false, Input: "0 0\n", Output: "0\n", Description: "Zero values"},
			{Rank: 4, IsSample: false, Input: "1000000000 1000000000\n", Output: "2000000000\n", Description: "Large numbers"},
			{Rank: 5, IsSample: false, Input: "-1000000000 -1000000000\n", Output: "-2000000000\n", Description: "Large negative numbers"},
		},
	},
	{
		Name:        "Sum of Array",
		Difficulty:  "easy",
		TimeLimit:   1.0,
		MemoryLimit: 262144,
		Points:      100,
		ProblemStatement: `
<h1>Sum of Array</h1>
<h2>Problem Description</h2>
<p>Given an array of N integers, calculate the sum of all elements.</p>
<h2>Input</h2>
<p>The first line contains an integer N (1 ≤ N ≤ 1000), the number of elements in the array.</p>
<p>The second line contains N integers separated by spaces.</p>
<p>Constraints:</p>
<ul>
<li>Each element is between -10<sup>6</sup> and 10<sup>6</sup></li>
</ul>
<h2>Output</h2>
<p>Output a single integer representing the sum of all elements.</p>
<h2>Sample Input 1</h2>
<pre>5
1 2 3 4 5</pre>
<h2>Sample Output 1</h2>
<pre>15</pre>
<h2>Sample Input 2</h2>
<pre>3
-1 5 -3</pre>
<h2>Sample Output 2</h2>
<pre>1</pre>
`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "5\n1 2 3 4 5\n", Output: "15\n", Description: "Simple positive numbers"},
			{Rank: 2, IsSample: true, Input: "3\n-1 5 -3\n", Output: "1\n", Description: "Mixed numbers"},
			{Rank: 3, IsSample: false, Input: "1\n42\n", Output: "42\n", Description: "Single element"},
			{Rank: 4, IsSample: false, Input: "4\n1000000 -1000000 1000000 -1000000\n", Output: "0\n", Description: "Cancelling numbers"},
		},
	},
	{
		Name:        "Fibonacci Number",
		Difficulty:  "easy",
		TimeLimit:   1.0,
		MemoryLimit: 262144,
		Points:      100,
		ProblemStatement: `
<h1>Fibonacci Number</h1>
<h2>Problem Description</h2>
<p>The Fibonacci sequence is defined as follows:</p>
<ul>
<li>F(0) = 0</li>
<li>F(1) = 1</li>
<li>F(n) = F(n-1) + F(n-2) for n ≥ 2</li>
</ul>
<p>Given an integer n, calculate F(n).</p>
<h2>Input</h2>
<p>A single integer n (0 ≤ n ≤ 30).</p>
<h2>Output</h2>
<p>Output F(n).</p>
<h2>Sample Input 1</h2>
<pre>0</pre>
<h2>Sample Output 1</h2>
<pre>0</pre>
<h2>Sample Input 2</h2>
<pre>10</pre>
<h2>Sample Output 2</h2>
<pre>55</pre>
<h2>Hint</h2>
<p>You can use recursion, iteration, or dynamic programming.</p>
`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "0\n", Output: "0\n", Description: "Base case F(0)"},
			{Rank: 2, IsSample: true, Input: "10\n", Output: "55\n", Description: "F(10)"},
			{Rank: 3, IsSample: false, Input: "1\n", Output: "1\n", Description: "Base case F(1)"},
			{Rank: 4, IsSample: false, Input: "20\n", Output: "6765\n", Description: "F(20)"},
			{Rank: 5, IsSample: false, Input: "30\n", Output: "832040\n", Description: "F(30)"},
		},
	},
	{
		Name:        "Binary Search",
		Difficulty:  "medium",
		TimeLimit:   1.0,
		MemoryLimit: 262144,
		Points:      200,
		ProblemStatement: `
<h1>Binary Search</h1>
<h2>Problem Description</h2>
<p>Given a sorted array of N distinct integers and a target value, find the index of the target in the array using binary search.</p>
<p>If the target is not found, output -1.</p>
<h2>Input</h2>
<p>The first line contains N (1 ≤ N ≤ 10<sup>5</sup>).</p>
<p>The second line contains N sorted integers separated by spaces.</p>
<p>The third line contains the target value to search for.</p>
<h2>Output</h2>
<p>Output the index of the target (0-based), or -1 if not found.</p>
<h2>Sample Input 1</h2>
<pre>5
1 3 5 7 9
5</pre>
<h2>Sample Output 1</h2>
<pre>2</pre>
<h2>Sample Input 2</h2>
<pre>5
1 3 5 7 9
4</pre>
<h2>Sample Output 2</h2>
<pre>-1</pre>
<h2>Hint</h2>
<p>Implement binary search with O(log N) time complexity.</p>
`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "5\n1 3 5 7 9\n5\n", Output: "2\n", Description: "Found in middle"},
			{Rank: 2, IsSample: true, Input: "5\n1 3 5 7 9\n4\n", Output: "-1\n", Description: "Not found"},
			{Rank: 3, IsSample: false, Input: "1\n42\n42\n", Output: "0\n", Description: "Single element found"},
			{Rank: 4, IsSample: false, Input: "1\n42\n41\n", Output: "-1\n", Description: "Single element not found"},
			{Rank: 5, IsSample: false, Input: "10\n1 2 3 4 5 6 7 8 9 10\n1\n", Output: "0\n", Description: "First element"},
			{Rank: 6, IsSample: false, Input: "10\n1 2 3 4 5 6 7 8 9 10\n10\n", Output: "9\n", Description: "Last element"},
		},
	},
	{
		Name:        "Two Sum",
		Difficulty:  "medium",
		TimeLimit:   1.0,
		MemoryLimit: 262144,
		Points:      200,
		ProblemStatement: `
<h1>Two Sum</h1>
<h2>Problem Description</h2>
<p>Given an array of N integers and a target sum, find two numbers in the array that add up to the target.</p>
<p>Return their indices (0-based). There is exactly one solution, and you cannot use the same element twice.</p>
<h2>Input</h2>
<p>The first line contains N (2 ≤ N ≤ 10<sup>4</sup>).</p>
<p>The second line contains N integers.</p>
<p>The third line contains the target sum.</p>
<h2>Output</h2>
<p>Output two indices i and j (i < j) separated by a space.</p>
<h2>Sample Input 1</h2>
<pre>4
2 7 11 15
9</pre>
<h2>Sample Output 1</h2>
<pre>0 1</pre>
<h2>Sample Input 2</h2>
<pre>3
3 2 4
6</pre>
<h2>Sample Output 2</h2>
<pre>1 2</pre>
<h2>Hint</h2>
<p>Use a hash map for O(N) time complexity.</p>
`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "4\n2 7 11 15\n9\n", Output: "0 1\n", Description: "First two elements"},
			{Rank: 2, IsSample: true, Input: "3\n3 2 4\n6\n", Output: "1 2\n", Description: "Middle elements"},
			{Rank: 3, IsSample: false, Input: "2\n1 1\n2\n", Output: "0 1\n", Description: "Duplicate values"},
			{Rank: 4, IsSample: false, Input: "5\n-1 -2 -3 -4 -5\n-8\n", Output: "2 4\n", Description: "Negative numbers"},
		},
	},
	{
		Name:        "Longest Common Subsequence",
		Difficulty:  "hard",
		TimeLimit:   2.0,
		MemoryLimit: 524288, // 512 MB
		Points:      300,
		ProblemStatement: `
<h1>Longest Common Subsequence</h1>
<h2>Problem Description</h2>
<p>Given two strings, find the length of their longest common subsequence (LCS).</p>
<p>A subsequence is a sequence that can be derived from another sequence by deleting some elements without changing the order of the remaining elements.</p>
<h2>Input</h2>
<p>The first line contains string S1.</p>
<p>The second line contains string S2.</p>
<p>Constraints:</p>
<ul>
<li>1 ≤ |S1|, |S2| ≤ 1000</li>
<li>Both strings contain only lowercase letters.</li>
</ul>
<h2>Output</h2>
<p>Output the length of the LCS.</p>
<h2>Sample Input 1</h2>
<pre>abcde
ace</pre>
<h2>Sample Output 1</h2>
<pre>3</pre>
<h2>Sample Input 2</h2>
<pre>abc
abc</pre>
<h2>Sample Output 2</h2>
<pre>3</pre>
<h2>Hint</h2>
<p>Use dynamic programming. Create a 2D table where dp[i][j] represents the LCS length of S1[0..i] and S2[0..j].</p>
`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "abcde\nace\n", Output: "3\n", Description: "LCS 'ace'"},
			{Rank: 2, IsSample: true, Input: "abc\nabc\n", Output: "3\n", Description: "Same strings"},
			{Rank: 3, IsSample: false, Input: "abc\ndef\n", Output: "0\n", Description: "No common"},
			{Rank: 4, IsSample: false, Input: "aaaaaaaaaa\naaaaa\n", Output: "5\n", Description: "Repeated characters"},
			{Rank: 5, IsSample: false, Input: "abcdgh\naedfhr\n", Output: "3\n", Description: "LCS 'adh'"},
		},
	},
	{
		Name:        "Shortest Path",
		Difficulty:  "hard",
		TimeLimit:   2.0,
		MemoryLimit: 262144,
		Points:      300,
		ProblemStatement: `
<h1>Shortest Path</h1>
<h2>Problem Description</h2>
<p>Given a weighted directed graph with N nodes and M edges, find the shortest path from node 1 to node N.</p>
<p>If there is no path, output -1.</p>
<h2>Input</h2>
<p>The first line contains N and M (2 ≤ N ≤ 10<sup>5</sup>, 1 ≤ M ≤ 10<sup>5</sup>).</p>
<p>The next M lines each contain three integers: u, v, w, representing an edge from u to v with weight w.</p>
<p>Constraints:</p>
<ul>
<li>1 ≤ u, v ≤ N</li>
<li>1 ≤ w ≤ 10<sup>9</sup></li>
<li>There are no negative weight edges.</li>
</ul>
<h2>Output</h2>
<p>Output the shortest distance from node 1 to node N, or -1 if unreachable.</p>
<h2>Sample Input 1</h2>
<pre>4 4
1 2 1
2 3 2
3 4 3
1 4 10</pre>
<h2>Sample Output 1</h2>
<pre>6</pre>
<h2>Sample Input 2</h2>
<pre>3 1
1 2 5</pre>
<h2>Sample Output 2</h2>
<pre>-1</pre>
<h2>Hint</h2>
<p>Use Dijkstra's algorithm with a priority queue for O((N+M) log N) time complexity.</p>
`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "4 4\n1 2 1\n2 3 2\n3 4 3\n1 4 10\n", Output: "6\n", Description: "Path 1->2->3->4"},
			{Rank: 2, IsSample: true, Input: "3 1\n1 2 5\n", Output: "-1\n", Description: "No path to node 3"},
			{Rank: 3, IsSample: false, Input: "2 1\n1 2 100\n", Output: "100\n", Description: "Direct edge"},
			{Rank: 4, IsSample: false, Input: "5 6\n1 2 3\n2 3 4\n3 4 5\n4 5 6\n1 5 100\n2 5 20\n", Output: "18\n", Description: "Complex graph"},
		},
	},
	{
		Name:        "Maximum Subarray",
		Difficulty:  "medium",
		TimeLimit:   1.0,
		MemoryLimit: 262144,
		Points:      200,
		ProblemStatement: `
<h1>Maximum Subarray</h1>
<h2>Problem Description</h2>
<p>Given an array of N integers, find the contiguous subarray with the largest sum.</p>
<h2>Input</h2>
<p>The first line contains N (1 ≤ N ≤ 10<sup>5</sup>).</p>
<p>The second line contains N integers.</p>
<p>Each integer is between -10<sup>4</sup> and 10<sup>4</sup>.</p>
<h2>Output</h2>
<p>Output the maximum sum of a contiguous subarray.</p>
<h2>Sample Input 1</h2>
<pre>9
-2 1 -3 4 -1 2 1 -5 4</pre>
<h2>Sample Output 1</h2>
<pre>6</pre>
<h2>Sample Input 2</h2>
<pre>1
-1</pre>
<h2>Sample Output 2</h2>
<pre>-1</pre>
<h2>Hint</h2>
<p>Use Kadane's algorithm for O(N) time complexity.</p>
`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "9\n-2 1 -3 4 -1 2 1 -5 4\n", Output: "6\n", Description: "Subarray [4,-1,2,1]"},
			{Rank: 2, IsSample: true, Input: "1\n-1\n", Output: "-1\n", Description: "Single negative"},
			{Rank: 3, IsSample: false, Input: "5\n1 2 3 4 5\n", Output: "15\n", Description: "All positive"},
			{Rank: 4, IsSample: false, Input: "3\n-1 -2 -3\n", Output: "-1\n", Description: "All negative"},
		},
	},
	{
		Name:        "Palindrome Check",
		Difficulty:  "easy",
		TimeLimit:   1.0,
		MemoryLimit: 262144,
		Points:      100,
		ProblemStatement: `
<h1>Palindrome Check</h1>
<h2>Problem Description</h2>
<p>Given a string, determine if it is a palindrome. A palindrome reads the same forward and backward.</p>
<p>Ignore spaces and consider only alphanumeric characters. Ignore case differences.</p>
<h2>Input</h2>
<p>A single line containing the string (length ≤ 1000).</p>
<h2>Output</h2>
<p>Output "true" if the string is a palindrome, "false" otherwise.</p>
<h2>Sample Input 1</h2>
<pre>A man a plan a canal Panama</pre>
<h2>Sample Output 1</h2>
<pre>true</pre>
<h2>Sample Input 2</h2>
<pre>race a car</pre>
<h2>Sample Output 2</h2>
<pre>false</pre>
`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "A man a plan a canal Panama\n", Output: "true\n", Description: "Classic palindrome"},
			{Rank: 2, IsSample: true, Input: "race a car\n", Output: "false\n", Description: "Not palindrome"},
			{Rank: 3, IsSample: false, Input: "hello\n", Output: "false\n", Description: "Simple word"},
			{Rank: 4, IsSample: false, Input: "Was it a car or a cat I saw\n", Output: "true\n", Description: "Another palindrome"},
		},
	},
	{
		Name:        "Prime Factorization",
		Difficulty:  "medium",
		TimeLimit:   1.0,
		MemoryLimit: 262144,
		Points:      200,
		ProblemStatement: `
<h1>Prime Factorization</h1>
<h2>Problem Description</h2>
<p>Given a positive integer N, find its prime factorization.</p>
<p>Output the prime factors in increasing order, with their exponents.</p>
<h2>Input</h2>
<p>A single integer N (2 ≤ N ≤ 10<sup>12</sup>).</p>
<h2>Output</h2>
<p>Output each prime factor and its exponent, separated by a space, one per line.</p>
<h2>Sample Input 1</h2>
<pre>12</pre>
<h2>Sample Output 1</h2>
<pre>2 2
3 1</pre>
<h2>Sample Input 2</h2>
<pre>100</pre>
<h2>Sample Output 2</h2>
<pre>2 2
5 2</pre>
<h2>Hint</h2>
<p>Use trial division up to sqrt(N).</p>
`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "12\n", Output: "2 2\n3 1\n", Description: "12 = 2^2 * 3"},
			{Rank: 2, IsSample: true, Input: "100\n", Output: "2 2\n5 2\n", Description: "100 = 2^2 * 5^2"},
			{Rank: 3, IsSample: false, Input: "2\n", Output: "2 1\n", Description: "Prime number"},
			{Rank: 4, IsSample: false, Input: "17\n", Output: "17 1\n", Description: "Prime number"},
			{Rank: 5, IsSample: false, Input: "1024\n", Output: "2 10\n", Description: "Power of 2"},
		},
	},
	{
		Name:             "Stack",
		Difficulty:       "easy",
		TimeLimit:        1.0,
		MemoryLimit:      262144,
		Points:           100,
		ProblemStatement: `<h1>Stack Implementation</h1><p>Implement a stack.</p>`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "5\npush 10\ntop\n", Output: "10\n", Description: "test"},
		},
	},
	{
		Name:             "String Match",
		Difficulty:       "medium",
		TimeLimit:        1.0,
		MemoryLimit:      262144,
		Points:           200,
		ProblemStatement: `<h1>String Match</h1><p>Find pattern.</p>`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "abab\nab\n", Output: "0 2\n", Description: "test"},
		},
	},
	{
		Name:             "Sorting",
		Difficulty:       "easy",
		TimeLimit:        2.0,
		MemoryLimit:      262144,
		Points:           100,
		ProblemStatement: `<h1>Sorting</h1><p>Sort array.</p>`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "3\n3 1 2\n", Output: "1 2 3\n", Description: "test"},
		},
	},
	{
		Name:             "Tree",
		Difficulty:       "hard",
		TimeLimit:        2.0,
		MemoryLimit:      524288,
		Points:           300,
		ProblemStatement: `<h1>Tree</h1><p>Tree traversal.</p>`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "3\n2 1 3\n1 2 3\n", Output: "2 3 1\n", Description: "test"},
		},
	},
	{
		Name:             "DFS",
		Difficulty:       "medium",
		TimeLimit:        1.0,
		MemoryLimit:      262144,
		Points:           200,
		ProblemStatement: `<h1>DFS</h1><p>DFS traversal.</p>`,
		TestCases: []SampleTestCase{
			{Rank: 1, IsSample: true, Input: "3 2\n1 2\n2 3\n", Output: "1 2 3\n", Description: "test"},
		},
	},
}

// Sample contest data
var sampleContests = []SampleContest{
	{
		ExternalID:   "intro-contest-2024",
		Name:         "Introduction to Algorithms",
		ShortName:    "INTRO2024",
		StartTime:    time.Now().Add(-24 * time.Hour),
		EndTime:      time.Now().Add(168 * time.Hour),
		ProblemNames: []string{"A + B", "Sum of Array", "Fibonacci Number", "Palindrome Check", "Stack", "Sorting"},
		ShortNames:   []string{"A", "B", "C", "D", "E", "F"},
		Points:       []int32{100, 100, 100, 100, 100, 100},
	},
	{
		ExternalID:   "advanced-contest-2024",
		Name:         "Advanced Algorithms",
		ShortName:    "ADV2024",
		StartTime:    time.Now().Add(-2 * time.Hour),
		EndTime:      time.Now().Add(120 * time.Hour),
		ProblemNames: []string{"Binary Search", "Two Sum", "Maximum Subarray", "Prime Factorization", "String Match", "DFS"},
		ShortNames:   []string{"A", "B", "C", "D", "E", "F"},
		Points:       []int32{200, 200, 200, 200, 200, 200},
	},
	{
		ExternalID:   "challenge-contest-2024",
		Name:         "Challenge Contest",
		ShortName:    "CHAL2024",
		StartTime:    time.Now().Add(1 * time.Hour),
		EndTime:      time.Now().Add(72 * time.Hour),
		ProblemNames: []string{"Longest Common Subsequence", "Shortest Path", "Tree"},
		ShortNames:   []string{"A", "B", "C"},
		Points:       []int32{300, 300, 300},
	},
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://oj:oj@localhost:5432/oj?sslmode=disable"
	}

	ctx := context.Background()

	// Connect to database
	dbpool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbpool.Close()

	log.Println("Connected to database")

	// Seed languages
	if err := seedLanguages(ctx, dbpool); err != nil {
		log.Fatalf("Failed to seed languages: %v", err)
	}
	log.Println("Seeded languages")

	// Seed problems
	problemIDMap := make(map[string]string)
	for _, problem := range sampleProblems {
		id, err := seedProblem(ctx, dbpool, problem)
		if err != nil {
			log.Printf("Failed to seed problem %s: %v", problem.Name, err)
			continue
		}
		problemIDMap[problem.Name] = id
		log.Printf("Seeded problem %s with ID %s", problem.Name, id)
	}

	// Seed contests
	for _, contest := range sampleContests {
		if err := seedContest(ctx, dbpool, contest, problemIDMap); err != nil {
			log.Printf("Failed to seed contest %s: %v", contest.ExternalID, err)
			continue
		}
		log.Printf("Seeded contest %s (%s)", contest.ExternalID, contest.Name)
	}

	log.Println("Seeding completed successfully!")
}

func seedLanguages(ctx context.Context, db *pgxpool.Pool) error {
	languages := []struct {
		ID         string
		ExternalID string
		Name       string
		TimeFactor float64
		Extensions []string
	}{
		{"cpp", "cpp", "C++ 17", 1.0, []string{".cpp", ".cc", ".cxx"}},
		{"c", "c", "C11", 1.0, []string{".c"}},
		{"python3", "python3", "Python 3", 2.0, []string{".py", ".py3"}},
		{"java", "java", "Java 17", 1.5, []string{".java"}},
		{"go", "go", "Go 1.21", 1.2, []string{".go"}},
		{"rust", "rust", "Rust", 1.0, []string{".rs"}},
		{"nodejs", "nodejs", "Node.js 18", 2.0, []string{".js", ".mjs"}},
	}

	for _, lang := range languages {
		extArray := fmt.Sprintf("ARRAY[%s]", strings.Join(quoteStrings(lang.Extensions), ", "))
		query := fmt.Sprintf(`
			INSERT INTO languages (id, external_id, name, time_factor, extensions, allow_submit, allow_judge)
			VALUES ($1, $2, $3, $4, %s, true, true)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				time_factor = EXCLUDED.time_factor,
				extensions = EXCLUDED.extensions
		`, extArray)

		_, err := db.Exec(ctx, query, lang.ID, lang.ExternalID, lang.Name, lang.TimeFactor)
		if err != nil {
			return err
		}
	}
	return nil
}

func quoteStrings(strs []string) []string {
	result := make([]string, len(strs))
	for i, s := range strs {
		result[i] = fmt.Sprintf("'%s'", s)
	}
	return result
}

func problemPathKey(name string) string {
	key := strings.ToLower(name)
	key = strings.NewReplacer("+", " plus ").Replace(key)

	var b strings.Builder
	lastDash := false
	for _, r := range key {
		isLetter := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isLetter || isDigit {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}

	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "problem"
	}
	return out
}

func seedProblem(ctx context.Context, db *pgxpool.Pool, problem SampleProblem) (string, error) {
	var idBytes [16]byte
	err := db.QueryRow(ctx, `
		SELECT id
		FROM problems
		WHERE name = $1
		LIMIT 1
	`, problem.Name).Scan(&idBytes)
	if err != nil {
		if err != pgx.ErrNoRows {
			return "", err
		}

		insertQuery := `
			INSERT INTO problems (name, time_limit, memory_limit, output_limit, difficulty, points, is_published, allow_submit)
			VALUES ($1, $2, $3, $4, $5, $6, true, true)
			RETURNING id
		`
		err = db.QueryRow(ctx, insertQuery,
			problem.Name, problem.TimeLimit, problem.MemoryLimit,
			4096, problem.Difficulty, problem.Points,
		).Scan(&idBytes)
		if err != nil {
			return "", err
		}
	} else {
		existingProblemID := uuid.UUID(idBytes).String()
		_, err = db.Exec(ctx, `
			UPDATE problems
			SET time_limit = $2,
				memory_limit = $3,
				output_limit = $4,
				difficulty = $5,
				points = $6,
				is_published = true,
				allow_submit = true,
				updated_at = NOW()
			WHERE id = $1
		`, existingProblemID, problem.TimeLimit, problem.MemoryLimit, 4096, problem.Difficulty, problem.Points)
		if err != nil {
			return "", err
		}
	}

	problemID := uuid.UUID(idBytes).String()

	// Insert problem statement
	stmtQuery := `
		INSERT INTO problem_statements (problem_id, language, format, title, content)
		VALUES ($1, 'en', 'html', $2, $3)
		ON CONFLICT (problem_id, language) DO UPDATE SET content = EXCLUDED.content
	`
	_, err = db.Exec(ctx, stmtQuery, problemID, problem.Name, problem.ProblemStatement)
	if err != nil {
		log.Printf("Warning: Failed to insert problem statement for %s: %v", problem.Name, err)
	}

	// Insert test cases
	for _, tc := range problem.TestCases {
		tcQuery := `
			INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (problem_id, rank) DO UPDATE SET
				input_content = EXCLUDED.input_content,
				output_content = EXCLUDED.output_content,
				description = EXCLUDED.description
		`
		pathKey := problemPathKey(problem.Name)
		inputPath := fmt.Sprintf("problems/%s/input/test%d.txt", pathKey, tc.Rank)
		outputPath := fmt.Sprintf("problems/%s/output/test%d.txt", pathKey, tc.Rank)

		// Always include content inline for judge to use
		inputContent := tc.Input
		outputContent := tc.Output

		_, err := db.Exec(ctx, tcQuery,
			problemID, tc.Rank, tc.IsSample, inputPath, outputPath,
			inputContent, outputContent, tc.Description,
		)
		if err != nil {
			log.Printf("Warning: Failed to insert test case %d for %s: %v", tc.Rank, problem.Name, err)
		}
	}

	return problemID, nil
}

func seedContest(ctx context.Context, db *pgxpool.Pool, contest SampleContest, problemIDMap map[string]string) error {
	// Insert contest
	query := `
		INSERT INTO contests (external_id, name, short_name, start_time, end_time, public)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (external_id) DO UPDATE SET
			name = EXCLUDED.name,
			short_name = EXCLUDED.short_name,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time
		RETURNING id
	`

	var idBytes [16]byte
	err := db.QueryRow(ctx, query,
		contest.ExternalID, contest.Name, contest.ShortName,
		contest.StartTime, contest.EndTime,
	).Scan(&idBytes)
	if err != nil {
		return err
	}

	contestID := uuid.UUID(idBytes).String()

	// Insert contest problems
	for i, problemName := range contest.ProblemNames {
		problemID := problemIDMap[problemName]
		if problemID == "" {
			log.Printf("Warning: Problem %s not found for contest %s", problemName, contest.ExternalID)
			continue
		}

		cpQuery := `
			INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
			VALUES ($1, $2, $3, $4, $5, true)
			ON CONFLICT (contest_id, problem_id) DO UPDATE SET
				short_name = EXCLUDED.short_name,
				rank = EXCLUDED.rank,
				points = EXCLUDED.points
		`
		_, err := db.Exec(ctx, cpQuery,
			contestID, problemID, contest.ShortNames[i], i+1, contest.Points[i],
		)
		if err != nil {
			log.Printf("Warning: Failed to add problem %s to contest %s: %v", problemName, contest.ExternalID, err)
		}
	}

	return nil
}
