import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { App } from "./App";
import { AppProviders } from "./providers/AppProviders";

describe("App", () => {
  it("renders the start conversation call to action", () => {
    render(
      <AppProviders>
        <MemoryRouter>
          <App />
        </MemoryRouter>
      </AppProviders>,
    );

    expect(
      screen.getByRole("heading", { name: "Start a conversation" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Start conversation" }),
    ).toBeInTheDocument();
  });
});
