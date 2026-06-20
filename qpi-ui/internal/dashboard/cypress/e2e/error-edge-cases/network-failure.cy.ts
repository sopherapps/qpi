describe("Error & Edge Cases — Network Failure Handling", () => {
  beforeEach(() => {
    cy.on("uncaught:exception", () => false);
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");
  });

  it("shows an alert when dismissing a notification fails", () => {
    // Mock a notification so the Clear All button is visible
    cy.intercept("GET", "/api/collections/notifications/records*", {
      statusCode: 200,
      body: {
        items: [
          {
            id: "fake-notification-id",
            title: "Test Alert",
            description: "Fake",
          },
        ],
      },
    }).as("getNotifications");

    cy.visit("/"); // Reload to ensure the mocked notification is loaded
    cy.wait("@getNotifications");

    // Intercept the dismiss endpoint and force a failure
    cy.intercept("POST", "/api/notifications/*/dismiss", {
      forceNetworkError: true,
    }).as("dismissFail");

    cy.on("window:alert", (txt) => {
      expect(txt).to.include("Dismiss failed");
    });

    // Try to dismiss via the header bell dropdown
    cy.get('button svg.lucide-bell').parent().click();
    cy.contains("button", "Clear All").click();
  });
});
