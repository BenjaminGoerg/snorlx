import { render, screen } from "@testing-library/react";
import { AboutCard } from "./AboutCard";

describe("AboutCard", () => {
  it("shows the frontend build version and the GitHub release", () => {
    render(<AboutCard />);

    expect(screen.getByRole("heading", { name: "About" })).toBeInTheDocument();
    expect(screen.getByText(__APP_VERSION__)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /github release/i })).toHaveAttribute(
      "href",
      `https://github.com/banshee86vr/snorlx/releases/tag/v${__APP_VERSION__}`,
    );
  });
});
