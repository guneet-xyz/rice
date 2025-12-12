return {
  'm4xshen/hardtime.nvim',
  enabled = false,
  lazy = false,
  dependencies = { 'MunifTanjim/nui.nvim' },
  opts = {
    restriction_mode = 'hint',
    disable_mouse = false,
    disabled_keys = {
      -- scrolling
      ['<Up>'] = false,
      ['<Down>'] = false,
    },
  },
}
