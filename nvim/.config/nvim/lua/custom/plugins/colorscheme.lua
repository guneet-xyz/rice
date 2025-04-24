return {
  'xiyaowong/transparent.nvim',
  dependencies = {
    'folke/tokyonight.nvim',
    'catppuccin/nvim',
    'EdenEast/nightfox.nvim',
    'daschw/leaf.nvim',
  },
  priority = 1000, -- Make sure to load this before all the other start plugins.
  init = function()
    -- Load the colorscheme here.
    -- Like many other themes, this one has different styles, and you could load
    -- any other, such as 'tokyonight-storm', 'tokyonight-moon', or 'tokyonight-day'.
    -- vim.cmd.colorscheme 'tokyonight-night'
    -- vim.cmd.colorscheme 'retrobox'
    --

    vim.o.wildignore = table.concat({
      -- A list of whatever else you want to ignore. E.g., "__pycache__", "*.o", etc.
      'blue.vim',
      'darkblue.vim',
      'delek.vim',
      'desert.vim',
      'elflord.vim',
      'evening.vim',
      'habamax.vim',
      'industry.vim',
      'koehler.vim',
      'lunaperche.vim',
      'morning.vim',
      'murphy.vim',
      'pablo.vim',
      'peachpuff.vim',
      'quiet.vim',
      'retrobox.vim',
      'ron.vim',
      'shine.vim',
      'slate.vim',
      'sorbet.vim',
      'torte.vim',
      'wildcharm.vim',
      'zaibatsu.vim',
      'zellner.vim',
    }, ',')
    -- vim.cmd.colorscheme 'retrobox'
    vim.cmd.colorscheme 'leaf'

    -- You can configure highlights by doing something like:
    vim.cmd.hi 'Comment gui=none'
  end,
}
