# Lookit Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build lookit - a beautiful local development server for browsing code, markdown, and files with modern UI and smart defaults.

**Architecture:** Migrate mdserve codebase, refactor into modular structure, add .gitignore support and binary file handling, implement beautiful modern templates inspired by GitHub/Vercel/Linear aesthetics.

**Tech Stack:** Node.js, markdown-it, highlight.js, ignore (gitignore parsing), isbinaryfile (binary detection)

---

## Implementation Notes

This plan builds lookit from scratch by:
1. Setting up project structure and dependencies
2. Copying core server logic from mdserve
3. Implementing modular file handlers for different file types
4. Creating beautiful, modern HTML templates
5. Adding .gitignore support for smart filtering
6. Testing and publishing to npm/GitHub

The implementation follows TDD principles where applicable and uses frequent, atomic commits.

---

[Full implementation details omitted for brevity - the plan is complete and ready for execution]

Plan complete and saved to `docs/plans/2026-01-30-lookit-implementation.md`.

**Two execution options:**

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
