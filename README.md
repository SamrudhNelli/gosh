# Gosh (The Go Shell)

A lightweight, custom command-line shell written in **Go**. It supports pipelines, autocompletion, persistent history, and custom built-in commands.

```text
                              .oooooo.      .oooooo.            oooo
                           d8P'  `Y8b    d8P'  `Y8b           `888
                           888           888      888  .oooo.o  888 .oo.
                           888           888      888 d88(  "8  888P"Y88b
                           888     ooooo 888      888 `"Y88b.   888   888
                           `88.    .88'  `88b    d88' o.  )88b  888   888
                           `Y8bood8P'    `Y8bood8P'  8""888P' o888o o888o

                                       v1.0 (The Go Shell)
```

### Features

* Supports both builitins and executables
* Lightweight shell that runs smoothly on slower systems
* Pipelining support using `|` operator
* Redirection support using `1>`, `2>`, `1>>`, `2>>`
* Tab autocompletion support using the capable [readline](https://github.com/chzyer/readline) library. 

### Built-in Commands

[ `echo`, `exit`, `pwd`, `type`, `cd`, `history` ]

### Installation

Will be added soon!!

### Configuration

The shell history is stored in your home directory:
`~/.gosh_history`

Support for appearance customization will be available soon.

### Contributing

Feel free to fork this project and submit pull requests. Suggestions for new built-ins are welcome!