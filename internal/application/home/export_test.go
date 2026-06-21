package home

// ClassifyForTest exposes the internal classify function for white-box tests
// in the home_test package. This file is excluded from production builds.
var ClassifyForTest = classify
