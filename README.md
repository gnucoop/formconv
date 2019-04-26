xls2ajf compiles [xlsform](http://xlsform.org/en/) excel files to ajf, a json-based format used at gnucoop to describe forms.
The tool can be installed with:

```go get bitbucket.org/gnucoop/xls2ajf```

and used as:

```xls2ajf form1.xls form2.xls form3.xlsx```

xls2ajf implements a subset of the xlsform specification.
Implemented features are listed in this document.

## Introduction to xlsforms

[Xlsform](http://xlsform.org/en/) is a standard that allows authoring forms in excel.
A xlsform excel file has two main sheets: "survey" and "choices".
The survey sheet describes the content of the form, while "choices" is used to define answers for single- or multiple-choice questions.
Empty rows and columns are ignored.
A simple example is given below.

Survey sheet:

|type                     |name       |label      |
|-------------------------|-----------|-----------|
|begin group              |info       |General Information |
|text                     |username   |Your name: |
|select_one yes_no        |pizza      |Do you like pizza? |
|select_multiple mealtime |mealtimes  |When do you have pizza? |
|end group                |           |           |

Choices sheet:

|list name |name      |label     |
|----------|----------|----------|
|yes_no    |yes       |Yes       |
|yes_no    |no        |No        |
|mealtime  |breakfast |Breakfast |
|mealtime  |lunch     |Lunch     |
|mealtime  |dinner    |Dinner    |

## Question types

The following table lists the supported question types.

|Question type   |ajf field type  |Description     |
|----------------|----------------|----------------|
|decimal         |number          |64-bit floating point number |
|text            |string          |Free text response |
|select_one      |single choice   |Single choice answer |
|select_multiple |multiple choice |Multiple choice answer |
|note            |empty           |Inserts an HTML note in the form |
|date            |date input      |A date          |
|time            |time            |Time            |

## Grouping

Questions can be grouped, as shown in the introductory example; groups can be nested.

Ajf forms have the peculiarity of being organized in slides, which has implications on how groups are handled.
Top-level groups are translated to slides, while inner groups are translated to ajf group nodes.
When the form contains ungrouped questions, the whole form will be wrapped in a single group/slide.
