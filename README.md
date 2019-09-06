formconv compiles [xlsform](http://xlsform.org/en/) excel files to ajf, a json-based format used at gnucoop to describe forms.
The tool can be installed with:

```go get github.com/gnucoop/formconv```

and used as:

```formconv form1.xlsx form2.xls form3.xls```

formconv implements a subset of the xlsform specification.
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
|integer         |number          |A number with the added constraint of being an integer |
|text            |string          |Free text response |
|boolean         |boolean         |Boolean answer (a checkbox) |
|select_one      |single choice   |Single choice answer |
|select_multiple |multiple choice |Multiple choice answer |
|note            |empty           |Inserts an HTML note in the form |
|date            |date input      |A date          |
|time            |time            |Time            |
|calculate       |formula         |Perform a [calculation](#calculation) |

## Required

It is possible to flag questions as required, so that the user won't be able to submit the form without providing a value:

|type      |name      |label     |required  |
|----------|----------|----------|----------|
|text      |color     |Your favorite color (very important information, mandatory): |yes |

## Grouping

Questions can be grouped, as shown in the [introductory example](#introduction-to-xlsforms); groups can be nested.

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

## Constraints

Constraints can be used to ensure data quality in the form:

|type      |name      |label            |constraint |constraint_message        |
|----------|----------|-----------------|-----------|--------------------------|
|integer   |age       |How old are you? |`. < 150`  |Age must be less than 150 |

The dot in the constraint formula refers to the value of the question.
The constraint message is optional.

## Relevant

The relevant column allows skipping a question or making and additional question appear based on the response to a previous question:

|type               |name      |label             |relevant             |
|-------------------|----------|------------------|---------------------|
|select_one cat_dog |pet_type  |Are you a cat or a dog person? |        |
|text               |cat_name  |Name of your cat: |`${pet_type} = "cat"`|
|text               |dog_name  |Name of your dog: |`${pet_type} = "dog"`|

The feature can also be applied to groups.

## Formulas

Formulas are used in the constraint, relevant and calculation columns.
formconv supports a subset of xlsform formulas.
In particular, the features involving nodesets are omitted, as ajf doesn't have an equivalent concept.

Formulas are expressions composed of constants, question references, operators and functions.
The expressions are parsed and converted to JavaScript.
No semantical analysis is done and the tool may produce invalid JavaScript like `45 + true`.

### Constants

Constants can be numbers, strings (delimited by 'single' or "double" quotes) or booleans (`True` or `False`).

### Question References

To reference the value provided as answer to a question, use the expression `${question_name}`.
The name must be a valid javascript identifier.
`.` can be used to refer to the current question, as seen in the [constraint example](#constraints).

### Operators

The following table lists the supported operators with their corresponding JavaScript implementation:

|                |   |   |   |     |     |     |     |   |    |   |    |     |    |
|----------------|---|---|---|-----|-----|-----|-----|---|----|---|----|-----|----|
| Formula op:    |`+`|`-`|`*`|`div`|`mod`|`=`  |`!=` |`>`|`>=`|`<`|`<=`|`and`|`or`|
| JavaScript op: |`+`|`-`|`*`|`/`  |`%`  |`===`|`!==`|`>`|`>=`|`<`|`<=`|`&&` |`ǀǀ`|

The precedence of operators is as defined by JavaScript operators.
Round parentheses can be used in formulas.

### Functions

#### String Manipulation Functions

|Formula function         |JavaScript translation |
|-------------------------|-----------------------|
|`regex(s, re)`           |`((s).match(re) !== null)`|
|`contains(s, t)`         |`(s).includes(t)`      |
|`starts-with(s, t)`      |`(s).startsWith(t)`    |
|`ends-with(s, t)`        |`(s).endsWith(t)`      |
|`substr(s, start[, end])`|`(s).substring(start[, end])`|
|`string-length(s)`       |`(s).length`           |
|`concat(s, t...)`        |`(s).concat(t...)`     |
|`string(x)`              |`String(x)`            |

#### Mathematical Functions

The following functions are available in formulas and are translated to the equivalent `Math` functions in JavaScript: `max`, `min`, `pow`, `log`, `log10`, `abs`, `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `atan2`, `sqrt`, `exp`, `random`.

Other functions dealing with numbers:

|Formula function |JavaScript/ajf translation |
|-----------------|---------------------------|
|`int(x)`         |`Math.floor(x)`            |
|`round(x, d)`    |`round(x, d)` (ajf function, rounds `x` to `d` digits) |
|`exp10(x)`       |`Math.pow(10, x)`          |
|`pi()`           |`Math.PI`                  |
|`number(x)`      |`Number(x)`                |

#### Boolean functions

|Formula function |JavaScript translation |
|-----------------|-----------------------|
|`not(x)`         |`!(x)`                 |
|`true()`         |`true`                 |
|`false()`        |`false`                |
|`boolean(x)`     |`Boolean(x)`           |

#### Other functions

|Formula function        |JavaScript/ajf translation |Description |
|------------------------|---------------------------|------------|
|`if(cond, then, else)`  |`(cond ? then : else)`     |            |
|`selected(${mul}, val)` |`valueInChoice(mul, val)`  |returns true if `val` has been selected <br> in the multiple choice question `mul` |
|`count-selected(${mul})`|`(mul).length`             |returns the number of options chosen <br> in the multiple choice question `mul` |

## Calculation

Calculations can be performed using the values of other questions:

|type      |name      |label               |calculation       |
|----------|----------|--------------------|------------------|
|decimal   |amount    |Price of your meal: |                  |
|calculate |tip       |5% tip is:          |`${amount} * 0.05`|

The results of calculations will appear as read-only fields in the form.

## Multiple language support

A form may include multiple languages with the following syntax:

|type      |name      |label::English (en) |label::Español (es)   |
|----------|----------|--------------------|----------------------|
|integer   |age       |How old are you?    |¿Cuántos años tienes? |
