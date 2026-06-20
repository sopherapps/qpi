describe("Error & Edge Cases — Network Failure Handling", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.get('input[type="text"]').type("user@example.com");
    cy.get('input[type="password"]').type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");
  });

  it("shows an alert when dismissing a notification fails", () => {
    // Intercept the dismiss endpoint and force a failure
    cy.intercept("POST", "/api/notifications/*/dismiss", {
      forceNetworkError: true,
    }).as("dismissFail");

    cy.on("window:alert", (txt) => {
      expect(txt).to.include("Dismiss failed");
    });

    // Try to dismiss via the header bell dropdown
    cy.get('button[aria-label="Notifications"]').click();
    cy.get("div").contains("Clear All").click();
  });
});
