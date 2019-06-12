xls2ajf compiles [xlsform](http://xlsform.org/en/) excel files to ajf, a json-based format used at gnucoop to describe forms.
The tool can be installed with:

```go get bitbucket.org/gnucoop/xls2ajf```

and used as:

```xls2ajf form1.xlsx form2.xls form3.xls```

xls2ajf implements a subset of the xlsform specification.
Supported features are listed in this document.

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

|Question type   |Ajf field type  |Description     |
|----------------|----------------|----------------|
|decimal         |number          |64-bit floating point number |
|text            |string          |Free text response |
|boolean         |boolean         |Boolean answer (a checkbox) |
|select_one      |single choice   |Single choice answer |
|select_multiple |multiple choice |Multiple choice answer |
|note            |empty           |Inserts an HTML note in the form |
|date            |date input      |A date          |
|time            |time            |Time            |

## Required

It is possible to flag questions as required, so that the user won't be able to submit the form without providing a value:

|type      |name      |label     |required  |
|----------|----------|----------|----------|
|text      |color     |Your favorite color (very important information, mandatory): |yes |

## Grouping

Questions can be grouped, as shown in the [introductory example](#markdown-header-introduction-to-xlsforms); groups can be nested.

Ajf forms have the peculiarity of being organized in slides, which has implications on how groups are handled.
Top-level groups are translated to slides, while inner groups are translated to ajf group nodes.
When the form contains ungrouped questions, the whole form will be wrapped in a single group/slide.

## Repeats

Repeats give the user the possibility to repeat a group of questions:

|type         |name         |label        |repeat_count |
|-------------|-------------|-------------|-------------|
|begin repeat |child_repeat |Answer the following questions for each one of your childs |20 |
|text         |name         |Child's name |             |
|decimal      |birthweight  |Child's birthweight |      |
|end repeat   |             |             |             |

When specified, `repeat_count` defines an upper bound to how many times the group can be repeated.
Repeats cannot be nested inside other repeats or groups.

## Relevant

The relevant column allows skipping a question or making and additional question appear based on the response to a previous question:

|type               |name      |label             |relevant            |
|-------------------|----------|------------------|--------------------|
|select_one cat_dog |pet_type  |Are you a cat or a dog person? |       |
|text               |cat_name  |Name of your cat: |${pet_type} = "cat" |
|text               |dog_name  |Name of your dog: |${pet_type} = "dog" |

The feature can also be applied to groups.
