import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { App } from "./App";
import { AppProviders } from "./providers/AppProviders";

describe("App", () => {
  it("renders the frontend foundation shell", () => {
    render(
      <AppProviders>
        <MemoryRouter>
          <App />
        </MemoryRouter>
      </AppProviders>,
    );

    expect(
      screen.getByRole("heading", { name: "React app foundation" }),
    ).toBeInTheDocument();
  });
});
