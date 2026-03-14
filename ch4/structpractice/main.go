package main

import "fmt"

type Employee struct {
	Name string
	ID   int
}

func getEmployeeByID(id int, employees []Employee) Employee {
	for i := range employees {
		e := &employees[i]
		if e.ID == id {
			return *e
		}
	}
	return Employee{}
}

func main() {
	employee1 := Employee{"a", 1}
	employee2 := Employee{"b", 2}
	var employees = []Employee{employee1, employee2}
	ee := getEmployeeByID(2, employees)
	ee.Name = "c"
	// prints bc
	fmt.Print(employees[1].Name)
	fmt.Print(ee.Name)
}
