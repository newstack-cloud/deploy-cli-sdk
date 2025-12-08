module.exports = {
  extends: ["@commitlint/config-conventional"],
  rules: {
    "type-enum": [
      2,
      "always",
      [
        "fix",
        "revert",
        "wip",
        "feat",
        "chore",
        "ci",
        "docs",
        "style",
        "refactor",
        "perf",
        "test",
        "deps"
      ],
    ],
  },
};
