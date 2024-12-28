-- You can add your own plugins here or in other files in this directory!
--  I promise not to create any merge conflicts in this directory :)
--
-- See the kickstart.nvim README for more information
return {

  'romgrk/barbar.nvim',
  lazy = false,
  dependencies = {
    'lewis6991/gitsigns.nvim', -- OPTIONAL: for git status
    'nvim-tree/nvim-web-devicons', -- OPTIONAL: for file icons
  },
  init = function()
    vim.g.barbar_auto_setup = false
  end,
  opts = {
    -- lazy.nvim will automatically call setup for you. put your options here, anything missing will use the default:
    -- animation = true,
    -- insert_at_start = true,
    -- …etc.
  },
  version = '^1.0.0', -- optional: only update when a new 1.x version is released
  keys = {
    { '<C-1>', ':BufferGoto 1<CR>' },
    { '<C-2>', ':BufferGoto 2<CR>' },
    { '<C-3>', ':BufferGoto 3<CR>' },
    { '<C-4>', ':BufferGoto 4<CR>' },
    { '<C-5>', ':BufferGoto 5<CR>' },
    { '<C-6>', ':BufferGoto 6<CR>' },
    { '<C-7>', ':BufferGoto 7<CR>' },
    { '<C-8>', ':BufferGoto 8<CR>' },
    { '<C-9>', ':BufferGoto 9<CR>' },
    { '<C-[>', ':BufferPrevious<CR>' },
    { '<C-]>', ':BufferNext<CR>' },
    { '<C-w>', ':BufferClose<CR>' },
  },
}
