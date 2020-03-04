# cfg_mngmnt the hardest to configure tool for server config management

To use:
`cfg_mngmnt --manifest <manifest.json>`

## Code flow

 1. Parse the manifest
 2. Register all actions for all units defined in the manifest
 3. Loop through sections in the order defined in the manifest
    1. Loop through units
        1. Executes `prerequisite` actions
        2. Executes `ensure` actions
        3. If `ensure` caused changes, executes `on-change` actions
        
 Executuing an action
 
   1. Execute the "verify" part of the action
   2. If the exit code for 0 was non zero, execute the "do" part of the action
   3. If the exit code for "do" was zero, execute the "verify" part of the action again
   
As long as exit code for all parts that were executed is 0, we're good. If not we
error.
   
If at any point an error occurs the problem dies. It doesn't have any way of
rolling back. A failed execution will result is a partially configured server.         

## Manifest

The manifest has three required components (granted that there is no validation
to ensure this):

* `actions` -- actions defined for a section
* `process-sections` -- the order in which to process sections
* `sections` -- the sections themselves

### Actions

Example of actions

    "actions": {
        "files": {
            "file-does-not-exist" : {
                "verify": "test ! -f ARG0",
                "do": "rm ARG0"
            },
            "file-exists" : {
                "args": [ "content" ],
                "verify": "test \"`echo 'ARG1' | md5sum | cut -d' ' -f1`\" = \"`md5sum ARG0 | cut -d' ' -f1`\"",
                "do": "echo 'ARG1' > ARG0"
            },...
    }
    
You define actions for a section. All units in the section will then "inherit" the action.

An action must have a `do`. If `verify` is defined it's used to check if we want to `do`, 
and again after `do` to make sure it worked.
If `verify` is not defined then `do` will always be executed.

`args` is an optional array of values which refer to `defintion` in a unit.

`ARG0` is the unit-name, `ARG1` is the first item in the args array, and so forth.

During the action registration phase actions for all units are parsed (all ARGs are replaced with 
relevant values) and storee in a global array.
The key to the action is section-name.unit-name.action-name.

### process-sections

    "process-sections": [
        "files"
    ],
    
This is simply an array of sections to process. They will be processed in the order defined.    

### sections

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
depends on how `add-packages.is-installed` and `services.restart` are defined in 
the `actions` component.

As long as section names match between `process-section`, `actions` and 
`sections` everything is fine.

The order in which sections will be processed is determined by `process-sections`.

### Unit

A section can contain 0 or more units. A **unit** has a

 * `name` -- this name is referred to as ARG0 in an action (required), though 
   nothing currently enforces this)
 * `prerequisite` -- a list of prerequisite actions
 * `ensure` -- a list of actions we want to ensure are try, it's here is
   where we'd make the changes
 * `on-change` -- a list of actions to perform if ensure caused any changes  
 * `definition` -- a list of meta data for the unit, when you define your actions
   you can refer to the meta data as arguments

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

 1. `prerequisite` actions are executed
 2. then `ensure` actions
 3. then, if `ensures` caused any change `on-change` actions will be executed

`prerequisite`, `ensure` and `on-change` are arrays of action references. 
An action reference can refer to an action to be executed on the current unit, 
another unit in the same section or a unit in a different section.

Action reference format
 * For current unit: action-name
 * For external unit: section-name.unit-name.action-name

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

## An example manifest

```
# In this manifest we have defined the following sections
#   packages
#   remove-packages
#   add-packages
#   files
#   remove-files
#   add-files
#   services

# packages and files are not included in process-manifest because we're
# using them to define actions to avoid having to define actions for everything
# in remove-packages and add-packages. Of course that means that any unit
# defined in remove-packages and add-packages has to exist in packages.
# actions contains only definitions for section packages and in the units
# in remove-packages and add-packages we therefore have to use the external
# type of action reference.

{
    "process-sections": [
        "remove-packages",
        "add-packages",
        "remove-files",
        "add-files",
        "services"
    ],

    "sections": {

# All actions defined for packages in actions will be registered for the units here.
    
        "packages": [ 
            { "name": "apache2" },
            { "name": "php5" },
            { "name": "zabbix-frontend-php" },
            { "name": "zend-framework" }
        ],
        "remove-packages": [ 
            {
                "name": "apache2",
                "prerequisite": [
                    "services.apache2.is-stopped"
                ],
                "ensure": [
                
# External reference to the actions defined for unit in the packages section.

                    "packages.apache2.is-not-installed"
                    
#  "is-not-installed" will not work because there are no actions defined for 
# remove-packages in the actions component.               
                    
                ]
            },
            {
                "name": "php5",
                "prerequisite": [
                    "packages.apache2.is-not-installed"
                ],
                "ensure": [
                    "packages.php5.is-not-installed"
                ]
            }
        ],
        "add-packages": [ 
            {
                "name": "apache2",
                "ensure": [
                    "packages.apache2.is-installed"
                ]
            },
            {
                "name": "php5",
                "prerequisite": [
                    "packages.apache2.is-installed"
                ],
                "ensure": [
                    "packages.php5.is-installed"
                ]
            },
            {
                "name": "zabbix-frontend-php",
                "prerequisite": [
                    "packages.apache2.is-installed",
                    "packages.php5.is-installed"
                ],
                "ensure": [
                    "packages.zabbix-frontend-php.is-installed"
                ],
                "on-change": [
                    "services.apache2.force-restart"
                ]
            },
            {
                "name": "zend-framework",
                "prerequisite": [
                    "packages.apache2.is-installed",
                    "packages.php5.is-installed"
                ],
                "ensure": [
                    "packages.zend-framework.is-installed"
                ],
                "on-change": [
                    "services.apache2.force-restart"
                ]
            }
        ],

        "services": [
            { 
                "name": "apache2",
                "ensure": [
                    "is-running"
                ]
            }
        ],
        
        "remove-files": [
            { 
                "name": "/var/www/html/info.php",
                "ensure": [
                    "files./var/www/html/info.php.file-does-not-exist"
                ]
            }
        ],

# Again we have a section which will we use only for action references.
        
        "files": [
            { 
                "name": "/var/www/html/info.php",
                "definition": {
                    "content": "<?php\nphpinfo();\n?>",
                    "owner": "root",
                    "group": "root",
                    "mode": "644"
                }
            },
            { 
                "name": "/var/www/html/info2.php",
                "definition": {
                    "content": "<?php\n\nheader(\"Content-Type: text/plain\");\n\necho \"Hello, world!\\n\";",
                    "owner": "root",
                    "group": "root",
                    "mode": "644"
                }
            }
        ],
        
        "add-files": [
            { 
                "name": "/var/www/html/info.php",
                "ensure": [
                    "files./var/www/html/info.php.file-exists",
                    "files./var/www/html/info.php.file-correct-mode"
                ]
            },
            { 
                "name": "/var/www/html/info2.php",
                "ensure": [
                    "files./var/www/html/info2.php.file-exists",
                    "files./var/www/html/info2.php.file-correct-mode"
                ]
            }
        ]
    },

# Non zero exit code means bad.
# Zero exit code means good.
# How well the manifest works depends on how well defined the actions are.
# A dry-run thing might be helpful since running a broken manifest will
# break a server, but alas.
# Debugging actions is kind of obnoxios.

    "actions": {
        "packages": {
            "is-installed": {
                "do": "apt-get update; apt-get --yes --force-yes install ARG0",
                "verify": "dpkg --get-selections | egrep \"^ARG0\\s+install\""
            },
            "is-not-installed": {
                "do": "apt-get --auto-remove --yes --force-yes purge ARG0",
                "verify": "test `/bin/false` -a `dpkg --get-selections | egrep \"^ARG0\\s+install\"`"
            }
        },
        "services": {
            "is-running": {
                "verify": "service ARG0 status | grep \"is running\"",
                "do": "service ARG0 restart"
            },
            "is-stopped": {
                "verify": "service ARG0 status 2>&1 | egrep \"is not running|unrecognized\"",
                "do": "service ARG0 stop"
            },
            "force-restart": {
                "do": "service ARG0 restart"
            }
        },
        
# It would be neat to be able to refer to build-in functions where do an verify actually referred
# to code code. 

        "files": {
            "file-does-not-exist" : {
                "verify": "test ! -f ARG0",
                "do": "rm ARG0"
            },
 
 # In this action content is a string in the manifest, but depending on how you write the action
 # it could be reference to another file and do could be to copy that file into place.
            
            "file-exists" : {
                "args": [ "content" ],
                "verify": "test -f ARG0 -a \"`echo 'ARG1' | md5sum | cut -d' ' -f1`\" = \"`md5sum ARG0 | cut -d' ' -f1`\"",
                "do": "echo 'ARG1' > ARG0"
            },
# Yeah.            
            
            "file-correct-mode" : {
                "args": [ "mode", "owner", "group" ],
                "do": "chmod ARG1 ARG0; chown ARG2:ARG3 ARG0",
                "verify": "test \"ARG1 ARG2 ARG3\" = \"`stat -c '%a %U %G' ARG0`\""
            }
        }
    }
}
```
