describe("Overview & Header — Notifications Panel", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').type("admin@example.com");
    cy.get('input[type="password"]').type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    // Broadcast an announcement so there's something to dismiss
    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");
    cy.contains("button", "Broadcast Announcement").click();

    cy.get('input[placeholder="QPU Maintenance Schedule"]')
      .type("Test Announcement");
    cy.get('textarea[placeholder="Rigetti Aspen-9 will be offline for calibration tomorrow..."]')
      .type("This is a test announcement for the notifications panel.");
    cy.get("form").contains("button", "Broadcast Announcement").click();

    cy.contains("Announcement broadcasted successfully!").should("be.visible");

    // Go back to Overview
    cy.contains("button", "Overview").click();
    cy.contains("h1", "Overview").should("be.visible");
  });

  it("dismisses an individual notification and it disappears from the panel", () => {
    // The notification should be visible in the System Announcements panel
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains("Test Announcement").should("be.visible");
      });

    // Hover over the notification to reveal the dismiss button, then click it
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains("Test Announcement")
          .parent()
          .parent()
          .parent()
          .trigger("mouseover")
          .within(() => {
            cy.get('button svg.lucide-x').parent().click();
          });
      });

    // The notification should disappear
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains("Test Announcement").should("not.exist");
      });
  });

  it('dismisses all notifications when "Clear All" is clicked from the header bell', () => {
    // Open the header bell dropdown
    cy.get('button svg.lucide-bell')
      .parent()
      .click();

    // Click "Clear All"
    cy.contains("button", "Clear All").click();

    // The System Announcements panel should show the empty state
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains("No active announcements").should("be.visible");
      });
  });
});
