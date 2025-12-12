# rice

These are my personal configurations for software's and tools i use on across multiple machines.

## Usage

> Preface: You are encouraged to make your own adjustments to all the configurations. Ricing is a very personal endeavor.

Most configurations are stow-compatible; with the [exceptions](#exceptions) being listed below. Read more about stow [here](https://www.gnu.org/software/stow/).

Example, to install the nvim config,

```
stow -v -t $HOME nvim
```

### Exceptions

Some configurations are not stow-compatible. These include:

- `ghostty` - I have some machine specific configurations for ghostty. Need to use the `install.sh` script in `ghostty` folder to install the configuration.
