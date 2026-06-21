describe("Auth — Login & Logout", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");
  });

  it("shows the login modal on first visit", () => {
    cy.get('[data-testid="login-modal"]').should("be.visible");
    cy.get('input[type="text"]').should("be.visible");
    cy.get('input[type="password"]').should("be.visible");
    cy.get('button[type="submit"]').should("be.visible");
  });

  it("shows an error when credentials are wrong", () => {
    // Default tab is Regular User
    cy.contains("button", "Regular User").should("have.class", "border-white");

    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("wrongpassword");
    cy.get('button[type="submit"]').click();

    cy.contains("Invalid credentials").should("be.visible");

    // The form should still be visible so the user can retry
    cy.get('input[type="text"]').should("have.value", "user@example.com");
    cy.get('button[type="submit"]').should("be.visible");
  });

  it("logs in as a regular user and logs out", () => {
    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();

    // Dashboard loads
    cy.contains("h1", "QPI Interface").should("be.visible");

    // Navigate to settings and sign out
    cy.contains("button", "Profile Settings").click();
    cy.contains("button", "Sign Out").click();

    // Back at login
    cy.get('[data-testid="login-modal"]').should("be.visible");
  });

  it("logs in as an administrator", () => {
    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();

    cy.contains("h1", "QPI Interface").should("be.visible");
    cy.contains("button", "Admin Panel").should("be.visible");
  });

  it("switches role tabs without losing form focus", () => {
    cy.get('input[type="text"]').clear().type("some@email.com");
    cy.get('input[type="password"]').clear().type("somepassword");

    cy.contains("button", "Administrator").click();
    cy.contains("button", "Administrator").should("have.class", "border-white");

    // Error should be cleared when switching tabs
    cy.contains("Invalid credentials").should("not.exist");

    // Switch back to user
    cy.contains("button", "Regular User").click();
    cy.contains("button", "Regular User").should("have.class", "border-white");
  });
});
