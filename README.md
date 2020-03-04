# cfg_mngmnt the hardest to configure tool for server config management

To use:
`cfg_mngmnt --manifest <manifest.json>`

## Manifest

The manifest has three required components (granted that there is no validation
to ensure this):

* `actions` -- actions defined for a all units in a section
* `process-sections` -- the order in which to process sections
* `sections` -- the sections themselves

### Sections

Example of sections

    "sections": {
        "section-name": [
            {
                {
                    "name": "unit-name",
                    "ensure": [ 
                        "action-name",                     
                        "section-name.unit-name.action-name" // This is the same as
                                                             // above, but also how
                                                             // you'd refer to an
                                                             // action on an external
                                                             // unit
                    ]
                },
                <unit2>,...
         ],
        "add-packages": [
                {
                    "name": "apache2",
                    "ensure": [ "is-installed" ]
                },
                {
                    "name": "php5",
                    "prerequisite": [ "add-packages.apache2.is-installed" ]
                    "ensure": [ "is-installed" ]
                    "on-change": [ "services.apache2.restart" ]
                },
                <unit3>,...
         ],...
    }

Unit is a json object described in more detail below.    

The section names are completely arbitrary, what happesn during `ensure`
depends on how you defined `add-packages.is-installed` and `services.
restart` are defined in the `actions` component.

The order in which sections will be processed is determined by `process-sections` 
and as long as section names match between `process-section`, `actions` and 
`sections` everything is fine.

There is no such thing as a type of section with special powers, what happens is
entirely determined by `actions`.

A section can contain 0 or more units. A unit has a

 * `name` -- this name is referred to as ARG0 in an action (required), though 
   nothing currently enforces this)
 * `prerequisite` -- a list of prerequisite actions
 * `ensure` -- a list of actions we want to ensure are try, it's here is
   where we'd make the changes
 * `on-change` -- a list of actions to perform if ensure caused any changes  
 * `definition` -- a list of meta data for the unit, when you define your actions
   you can refer to the meta data as arguments

Definition of a unit

A unit must have a `name`, but no other fields are required. An action can be executed
for a unit as long as it has a name.

    {
        "name": "unitname",
        "prerequisite": [],
        "ensure": [],
        "on-change": [],
        "definition": {
            "key1": "val1",
            "key2": "val2"
        }
   {


A section is an array of units, the units are processed sequentially.

Each unit are processed in the following way

 # `prerequisite` actions are executed
 # then `ensure` actions
 # if `ensures` caused any change then `on-change` actions will be executed

`prerequisite`, `ensure` and `on-change` can list actions which are defined
for the current unit or for another unit in the same section or for a unit
in a different section.

An action reference is action-name it it refers to an action defined for the unit
itself and section-name.unit-name.action-name if it refers to an action on 
an external unit.

If at any point an error occurs the problem dies. It doesn't have any way of
restoring a server to a pristine state in case of failure. A failed execution 
will result is a partially configured server.

Note that if a unit has an `ensure` with an `on-change`, but another eariler 
defined unit has defined the `ensure` as a `prerequite`, then the `on-change`
action will never happen. I.e.

    "sections": {
        "packages": [
            {
                "name": "apache2",
                "prerequisite": [
                    "packages.php5.is-installed"
                ],
                "ensure": [
                    "is-installed"
                ],
            },
            {
                "name": "php5",
                "on-change": [
                    "services.apache2.restart"
                ],
                "ensure": [
                    "is-installed"
                ],
            },...

In the above example php5 will be installed as a prerequisite for apache2 and
the `on-change` action in the php5 unit will never be executed.
