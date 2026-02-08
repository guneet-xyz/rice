return {
  'https://codeberg.org/andyg/leap.nvim',
  config = function()
    vim.keymap.set('n', '<space>', '<Plug>(leap)')
  end,
}
