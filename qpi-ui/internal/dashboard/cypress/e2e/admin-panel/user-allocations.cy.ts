describe("Admin Panel — User Allocations", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");
  });

  it("allocates QPU seconds to a user and updates the displayed quota", () => {
    // The "User Time Allocations" subtab is active by default
    cy.contains("button", "User Time Allocations")
      .should("have.class", "border-b-2");

    // Find the row for user@example.com and note the current quota
    cy.contains("td", "user@example.com")
      .parent("tr")
      .within(() => {
        cy.get("td").eq(2).invoke("text").as("initialQuota");
      });

    // Enter a new allocation amount and click Grant
    cy.contains("td", "user@example.com")
      .parent("tr")
      .within(() => {
        cy.get('input[type="number"]').clear().type("5000");
        cy.contains("button", "Grant").click();
      });

    // Verify the quota updates to the new value
    cy.contains("td", "user@example.com")
      .parent("tr")
      .within(() => {
        cy.get("td").eq(2).should("contain", "5000s");
      });
  });
});
